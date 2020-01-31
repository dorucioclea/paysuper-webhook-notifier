package internal

import (
	"bytes"
	"context"
	"fmt"
	"github.com/InVisionApp/go-health"
	"github.com/InVisionApp/go-health/handlers"
	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
	"github.com/micro/go-micro"
	"github.com/micro/go-plugins/client/selector/static"
	"github.com/paysuper/paysuper-proto/go/billingpb"
	"github.com/paysuper/paysuper-proto/go/recurringpb"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/handler"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	rabbitmq "gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serviceName   = "p1paynotifier"
	loggerName    = "PAYSUPER_WEBHOOK_NOTIFIER"
	mutexNameMask = "%s-%s"
)

type NotifierApplication struct {
	cfg  *config.Config
	repo billingpb.BillingService

	centrifugoPaymentForm handler.CentrifugoInterface
	centrifugoDashboard   handler.CentrifugoInterface

	httpServer *http.Server
	router     *http.ServeMux

	log                      *zap.Logger
	broker                   rabbitmq.BrokerInterface
	retryBroker              rabbitmq.BrokerInterface
	taxjarTransactionsBroker rabbitmq.BrokerInterface
	taxjarRefundsBroker      rabbitmq.BrokerInterface
	redis                    *redis.Client
}

type appHealthCheck struct {
	redis *redis.Client
}

type centrifugoHttpTransport struct {
	Transport http.RoundTripper
}

type centrifugoContextKey struct {
	name string
}

func NewApplication() *NotifierApplication {
	return &NotifierApplication{}
}

func NewCentrifugoHttpClient() *http.Client {
	return &http.Client{Transport: &centrifugoHttpTransport{}}
}

func (app *NotifierApplication) Init() {
	app.initLogger()
	app.initConfig()
	app.initRedis()
	app.initBroker()

	var service micro.Service

	options := []micro.Option{
		micro.Name(serviceName),
		micro.Version(recurringpb.PayOneMicroserviceVersion),
		micro.AfterStop(func() error {
			app.log.Info("Micro service stopped")
			return nil
		}),
	}

	if os.Getenv("MICRO_SELECTOR") == "static" {
		app.log.Info("Use micro selector `static`")
		options = append(options, micro.Selector(static.NewSelector()))
	}

	app.log.Info("Initialize micro service")

	service = micro.NewService(options...)
	service.Init()

	app.repo = billingpb.NewBillingService(billingpb.ServiceName, service.Client())
	app.centrifugoPaymentForm = handler.NewCentrifugo(app.cfg.CentrifugoPaymentForm, NewCentrifugoHttpClient())
	app.centrifugoDashboard = handler.NewCentrifugo(app.cfg.CentrifugoDashboard, NewCentrifugoHttpClient())

	app.router = http.NewServeMux()
	app.initHealth()
}

func (app *NotifierApplication) initRedis() {
	app.redis = redis.NewClient(&redis.Options{
		Addr:     app.cfg.RedisHost,
		Password: app.cfg.RedisPassword,
	})

	if _, err := app.redis.Ping().Result(); err != nil {
		zap.L().Fatal("Connection to Redis failed", zap.Error(err), zap.Any("options", app.cfg))
	}
}

func (app *NotifierApplication) initLogger() {
	logger, err := zap.NewProduction()

	if err != nil {
		log.Fatalf("Application logger initialization failed with error: %s\n", err)
	}

	app.log = logger.Named(loggerName)
	zap.ReplaceGlobals(app.log)
}

func (app *NotifierApplication) initConfig() {
	cfg, err := config.NewConfig()

	if err != nil {
		app.log.Fatal("Config init failed", zap.Error(err))
	}

	app.cfg = cfg
}

func (app *NotifierApplication) initBroker() {
	broker, err := rabbitmq.NewBroker(app.cfg.BrokerAddress)

	if err != nil {
		app.log.Fatal(
			"Creating RabbitMq broker failed",
			zap.Error(err),
			zap.String("amqp_url", app.cfg.BrokerAddress),
		)
	}

	retryBroker, err := rabbitmq.NewBroker(app.cfg.BrokerAddress)

	if err != nil {
		app.log.Fatal(
			"Creating RabbitMq retry broker failed",
			zap.Error(err),
			zap.String("amqp_url", app.cfg.BrokerAddress),
		)
	}

	retryBroker.(*rabbitmq.Broker).Opts.QueueOpts.Args = amqp.Table{
		"x-dead-letter-exchange":    recurringpb.PayOneTopicNotifyPaymentName,
		"x-message-ttl":             int32(handler.RetryDlxTimeout * 1000),
		"x-dead-letter-routing-key": "*",
	}
	retryBroker.SetExchangeName(handler.RetryExchangeName)

	err = broker.RegisterSubscriber(recurringpb.PayOneTopicNotifyPaymentName, app.Process)

	if err != nil {
		app.log.Fatal("Registration RabbitMQ broker handler failed", zap.Error(err))
	}

	taxjarTransactionsBroker, err := rabbitmq.NewBroker(app.cfg.BrokerAddress)
	if err != nil {
		app.log.Fatal(
			"Creating RabbitMq TaxJar transactions broker failed",
			zap.Error(err),
			zap.String("amqp_url", app.cfg.BrokerAddress),
		)
	}
	taxjarTransactionsBroker.SetExchangeName(recurringpb.TaxjarTransactionsTopicName)

	taxjarRefundsBroker, err := rabbitmq.NewBroker(app.cfg.BrokerAddress)
	if err != nil {
		app.log.Fatal(
			"Creating RabbitMq TaxJar transactions broker failed",
			zap.Error(err),
			zap.String("amqp_url", app.cfg.BrokerAddress),
		)
	}
	taxjarRefundsBroker.SetExchangeName(recurringpb.TaxjarRefundsTopicName)

	app.broker = broker
	app.retryBroker = retryBroker
	app.taxjarTransactionsBroker = taxjarTransactionsBroker
	app.taxjarRefundsBroker = taxjarRefundsBroker
}

func (app *NotifierApplication) initHealth() {
	h := health.New()
	err := h.AddChecks([]*health.Config{
		{
			Name: "health-check",
			Checker: &appHealthCheck{
				redis: app.redis,
			},
			Interval: time.Duration(1) * time.Second,
			Fatal:    true,
		},
	})

	if err != nil {
		app.log.Fatal("Health check register failed", zap.Error(err))
	}

	if err = h.Start(); err != nil {
		app.log.Fatal("Health check start failed", zap.Error(err))
	}

	app.log.Info("Health check listener started", zap.String("port", app.cfg.MetricsPort))

	app.router.HandleFunc("/health", handlers.NewJSONHandlerFunc(h, nil))
}

func (app *NotifierApplication) Run() {
	app.httpServer = &http.Server{
		Addr:    ":" + app.cfg.MetricsPort,
		Handler: app.router,
	}

	go func() {
		if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.log.Fatal("Http server starting failed", zap.Error(err))
		}
	}()

	app.log.Info("Http server started...")
	app.log.Info("Notifier started...")

	if err := app.broker.Subscribe(nil); err != nil {
		app.log.Fatal("Notifier subscriber start failed...", zap.Error(err))
	}
}

func (app *NotifierApplication) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.httpServer.Shutdown(ctx); err != nil {
		app.log.Fatal("Http server shutdown failed", zap.Error(err))
	}
	app.log.Info("Http server stopped")

	func() {
		if err := app.log.Sync(); err != nil {
			app.log.Fatal("Logger sync failed", zap.Error(err))
		} else {
			app.log.Info("Logger synced")
		}
	}()
}

func (app *NotifierApplication) Process(o *billingpb.Order, d amqp.Delivery) error {
	id := o.Id
	handlerName := o.Project.GetCallbackProtocol()
	mName := fmt.Sprintf(mutexNameMask, handlerName, id)

	mutex, err := lock.Obtain(app.redis, mName, nil)

	if err != nil {
		app.log.Error(err.Error())
		return err
	} else if mutex == nil {
		return nil
	}

	defer func() {
		if err := mutex.Unlock(); err != nil {
			app.log.Error("Mutex unlock failed", zap.Error(err))
		}
	}()

	h := handler.NewHandler(
		o,
		app.repo,
		app.retryBroker,
		app.taxjarTransactionsBroker,
		app.taxjarRefundsBroker,
		app.redis,
		d,
		app.cfg,
		app.centrifugoPaymentForm,
		app.centrifugoDashboard,
	)

	n, err := h.GetNotifier()

	if err != nil {
		return err
	}

	err = n.Notify()

	if h.RetryCount == 0 {
		err := h.SendToUserCentrifugo(o)

		if err != nil {
			h.HandleError(handler.LoggerNotificationCentrifugo, err, nil)
		}
	}

	return err
}

func (c *appHealthCheck) Status() (interface{}, error) {
	if _, err := c.redis.Ping().Result(); err != nil {
		return "fail", err
	}
	return "ok", nil
}

func (m *centrifugoHttpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), &centrifugoContextKey{name: "CentrifugoRequestStart"}, time.Now())
	req = req.WithContext(ctx)

	var reqBody []byte

	if req.Body != nil {
		reqBody, _ = ioutil.ReadAll(req.Body)
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))
	rsp, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		return rsp, err
	}

	var rspBody []byte

	if rsp.Body != nil {
		rspBody, err = ioutil.ReadAll(rsp.Body)

		if err != nil {
			return rsp, err
		}
	}

	rsp.Body = ioutil.NopCloser(bytes.NewBuffer(rspBody))
	req.Header.Set("Authorization", "apikey ****")

	zap.L().Info(
		req.URL.Path,
		zap.Any("request_headers", req.Header),
		zap.ByteString("request_body", reqBody),
		zap.Int("response_status", rsp.StatusCode),
		zap.Any("response_headers", rsp.Header),
		zap.ByteString("response_body", rspBody),
	)

	return rsp, err
}
