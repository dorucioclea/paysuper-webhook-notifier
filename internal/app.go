package internal

import (
	"context"
	"github.com/InVisionApp/go-health"
	"github.com/InVisionApp/go-health/handlers"
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/centrifugal/gocent"
	"github.com/micro/go-micro"
	k8s "github.com/micro/kubernetes/go/micro"
	"github.com/paysuper/paysuper-billing-server/pkg"
	proto "github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/handler"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
)

const (
	serviceName = "p1paynotifier"
)

type NotifierApplication struct {
	cfg    *config.Config
	repo   grpc.BillingService
	centCl *gocent.Client

	httpServer *http.Server
	router     *http.ServeMux

	log         *zap.Logger
	broker      *rabbitmq.Broker
	retryBroker *rabbitmq.Broker
}

type appHealthCheck struct{}

func NewApplication() *NotifierApplication {
	return &NotifierApplication{}
}

func (app *NotifierApplication) Init() {
	app.initLogger()
	app.initConfig()
	app.initBroker()

	var service micro.Service

	options := []micro.Option{
		micro.Name(serviceName),
		micro.Version(constant.PayOneMicroserviceVersion),
		micro.AfterStop(func() error {
			app.log.Info("Micro service stopped")
			return nil
		}),
	}

	if app.cfg.MicroRegistry == constant.RegistryKubernetes {
		service = k8s.NewService(options...)
		app.log.Info("[PAYSUPER_REPOSITORY] Initialize k8s service")
	} else {
		service = micro.NewService(options...)
		app.log.Info("[PAYSUPER_REPOSITORY] Initialize micro service")
	}

	service.Init()

	app.repo = grpc.NewBillingService(pkg.ServiceName, service.Client())
	app.centCl = gocent.New(
		gocent.Config{
			Addr:       app.cfg.CentrifugoUrl,
			Key:        app.cfg.CentrifugoKey,
			HTTPClient: tools.NewLoggedHttpClient(zap.S()),
		},
	)

	app.router = http.NewServeMux()
	app.initHealth()
}

func (app *NotifierApplication) initLogger() {
	var err error

	app.log, err = zap.NewProduction()

	if err != nil {
		log.Fatalf("Application logger initialization failed with error: %s\n", err)
	}
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
	retryBroker.Opts.QueueOpts.Args = amqp.Table{
		"x-dead-letter-exchange":    constant.PayOneTopicNotifyPaymentName,
		"x-message-ttl":             int32(handler.RetryDlxTimeout * 1000),
		"x-dead-letter-routing-key": "*",
	}
	retryBroker.Opts.ExchangeOpts.Name = handler.RetryExchangeName

	if err != nil {
		app.log.Fatal(
			"Creating RabbitMq retry broker failed",
			zap.Error(err),
			zap.String("amqp_url", app.cfg.BrokerAddress),
		)
	}

	err = broker.RegisterSubscriber(constant.PayOneTopicNotifyPaymentName, app.Process)

	if err != nil {
		app.log.Fatal("Registration RabbitMQ broker handler failed", zap.Error(err))
	}

	app.broker = broker
	app.retryBroker = retryBroker
}

func (app *NotifierApplication) initHealth() {
	h := health.New()
	err := h.AddChecks([]*health.Config{
		{
			Name:     "health-check",
			Checker:  &appHealthCheck{},
			Interval: time.Duration(1) * time.Second,
			Fatal:    true,
		},
	})

	if err != nil {
		app.log.Fatal("[PAYONE_BILLING] Health check register failed", zap.Error(err))
	}

	if err = h.Start(); err != nil {
		app.log.Fatal("[PAYONE_BILLING] Health check start failed", zap.Error(err))
	}

	app.log.Info("[PAYONE_BILLING] Health check listener started", zap.String("port", app.cfg.MetricsPort))

	app.router.HandleFunc("/health", handlers.NewJSONHandlerFunc(h, nil))
}

func (app *NotifierApplication) Run() {
	app.httpServer = &http.Server{
		Addr:    ":" + app.cfg.MetricsPort,
		Handler: app.router,
	}

	go func() {
		if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.log.Fatal("[PAYONE_BILLING] Http server starting failed", zap.Error(err))
		}
	}()

	app.log.Info("[PAYONE_NOTIFIER] Http server started...")
	app.log.Info("[PAYONE_NOTIFIER] Notifier started...")

	if err := app.broker.Subscribe(nil); err != nil {
		app.log.Fatal("[PAYONE_NOTIFIER] Notifier subscriber start failed...", zap.Error(err))
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

func (app *NotifierApplication) Process(o *proto.Order, d amqp.Delivery) error {
	h := handler.NewHandler(o, app.repo, app.centCl, app.retryBroker, d)

	if h.RetryCount == 0 && (o.Status == constant.OrderStatusPaymentSystemDeclined ||
		o.Status == constant.OrderStatusPaymentSystemCanceled) {
		if err := h.SendCentrifugoMessage(o); err != nil {
			h.HandleError(handler.LoggerNotificationCentrifugo, err, nil)
		}
		return nil
	}

	n, err := h.GetNotifier()

	if err != nil {
		return err
	}

	err = n.Notify()

	if h.RetryCount == 0 {
		if err := h.SendCentrifugoMessage(o); err != nil {
			h.HandleError(handler.LoggerNotificationCentrifugo, err, nil)
		}
	}

	return err
}

func (c *appHealthCheck) Status() (interface{}, error) {
	return "ok", nil
}
