package internal

import (
	"github.com/ProtocolONE/payone-notifier/internal/handler"
	"github.com/ProtocolONE/payone-repository/pkg/constant"
	proto "github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/ProtocolONE/payone-repository/pkg/proto/repository"
	"github.com/ProtocolONE/payone-repository/tools"
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/centrifugal/gocent"
	"github.com/kelseyhightower/envconfig"
	"github.com/micro/go-micro"
	k8s "github.com/micro/kubernetes/go/micro"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"log"
	"net/http"
)

const (
	serviceName = "p1paynotifier"
)

type Config struct {
	CentrifugoUrl  string `envconfig:"CENTRIFUGO_URL" required:"true"`
	CentrifugoKey  string `envconfig:"CENTRIFUGO_KEY" required:"true"`
	BrokerAddress  string `envconfig:"MICRO_BROKER_ADDRESS" required:"true"`
	KubernetesHost string `envconfig:"KUBERNETES_SERVICE_HOST" required:"false"`
}

type NotifierApplication struct {
	gRpcRepository   repository.RepositoryService
	SugaredLogger    *zap.SugaredLogger
	centrifugoClient *gocent.Client
	httpServer       *http.Server
	config           *Config

	Logger      *zap.Logger
	broker      *rabbitmq.Broker
	retryBroker *rabbitmq.Broker
}

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
	}

	if app.config.KubernetesHost == "" {
		service = micro.NewService(options...)
		app.SugaredLogger.Info("[PAYONE_NOTIFIER] Initialize micro service")
	} else {
		service = k8s.NewService(options...)
		app.SugaredLogger.Info("[PAYONE_NOTIFIER] Initialize k8s service")
	}

	service.Init()

	app.gRpcRepository = repository.NewRepositoryService(constant.PayOneRepositoryServiceName, service.Client())
	app.centrifugoClient = gocent.New(
		gocent.Config{
			Addr:       app.config.CentrifugoUrl,
			Key:        app.config.CentrifugoKey,
			HTTPClient: tools.NewLoggedHttpClient(app.SugaredLogger),
		},
	)
}

func (app *NotifierApplication) initLogger() {
	var err error

	app.Logger, err = zap.NewProduction()

	if err != nil {
		log.Fatalf("Application logger initialization failed with error: %s\n", err)
	}

	app.SugaredLogger = app.Logger.Sugar()
}

func (app *NotifierApplication) initConfig() {
	app.config = &Config{}

	if err := envconfig.Process("", app.config); err != nil {
		app.SugaredLogger.Fatalw("Config init failed", "error", err)
	}
}

func (app *NotifierApplication) initBroker() {
	broker, err := rabbitmq.NewBroker(app.config.BrokerAddress)

	if err != nil {
		app.SugaredLogger.Fatalw(
			"Creating RabbitMq broker failed",
			"error", err,
			"amqp_url", app.config.BrokerAddress,
		)
	}

	retryBroker, err := rabbitmq.NewBroker(app.config.BrokerAddress)
	retryBroker.Opts.QueueOpts.Args = amqp.Table{
		"x-dead-letter-exchange":    constant.PayOneTopicNotifyPaymentName,
		"x-message-ttl":             int32(handler.RetryDlxTimeout * 1000),
		"x-dead-letter-routing-key": "*",
	}
	retryBroker.Opts.ExchangeOpts.Name = handler.RetryExchangeName

	if err != nil {
		app.SugaredLogger.Fatalw(
			 "Creating RabbitMq retry broker failed",
			"error", err,
			"amqp_url", app.config.BrokerAddress,
		)
	}

	err = broker.RegisterSubscriber(constant.PayOneTopicNotifyPaymentName, app.Process)

	if err != nil {
		app.SugaredLogger.Fatalw("Registration RabbitMQ broker handler failed", "error", err)
	}

	app.broker = broker
	app.retryBroker = retryBroker
}

func (app *NotifierApplication) Run() {
	app.SugaredLogger.Info("[PAYONE_NOTIFIER] Notifier started...")

	if err := app.broker.Subscribe(nil); err != nil {
		app.SugaredLogger.Fatal(err)
	}
}

func (app *NotifierApplication) Process(o *proto.Order, d amqp.Delivery) error {
	h := handler.NewHandler(o, app.gRpcRepository, app.SugaredLogger, app.centrifugoClient, app.retryBroker, d)

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
