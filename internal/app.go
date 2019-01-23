package internal

import (
	"context"
	"github.com/ProtocolONE/payone-notifier/internal/handler"
	tools2 "github.com/ProtocolONE/payone-notifier/tools"
	"github.com/ProtocolONE/payone-repository/pkg/constant"
	proto "github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/ProtocolONE/payone-repository/pkg/proto/repository"
	"github.com/ProtocolONE/payone-repository/tools"
	"github.com/centrifugal/gocent"
	"github.com/kelseyhightower/envconfig"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
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
	serviceContext context.Context
	serviceCancel  context.CancelFunc

	service          micro.Service
	gRpcRepository   repository.RepositoryService
	sugaredLogger    *zap.SugaredLogger
	centrifugoClient *gocent.Client
	httpServer       *http.Server
	config           *Config

	Logger   *zap.Logger
	RabbitMq *tools2.RabbitMq
}

func NewApplication() *NotifierApplication {
	return &NotifierApplication{}
}

func (app *NotifierApplication) Init() {
	app.initConfig()

	options := []micro.Option{
		micro.Name(serviceName),
		micro.Version(constant.PayOneMicroserviceVersion),
	}

	if app.config.KubernetesHost == "" {
		app.service = micro.NewService(options...)
		log.Println("Initialize micro service")
	} else {
		app.service = k8s.NewService(options...)
		log.Println("Initialize k8s service")
	}

	app.service.Init()

	err := micro.RegisterSubscriber(constant.PayOneTopicNotifyPaymentName, app.service.Server(), app.Process, server.SubscriberQueue("queue.pubsub"))

	if err != nil {
		log.Fatal(err)
	}

	app.gRpcRepository = repository.NewRepositoryService(constant.PayOneRepositoryServiceName, app.service.Client())
	app.centrifugoClient = gocent.New(
		gocent.Config{
			Addr:       app.config.CentrifugoUrl,
			Key:        app.config.CentrifugoKey,
			HTTPClient: tools.NewLoggedHttpClient(app.sugaredLogger),
		},
	)

	app.RabbitMq = tools2.NewRabbitMq(app.config.BrokerAddress)
	app.RabbitMq.Opts.Queue.Args = amqp.Table{
		"x-message-ttl":             int32(10 * 1000),
		"x-dead-letter-exchange":    app.RabbitMq.Opts.Exchange.Name,
		"x-dead-letter-routing-key": app.RabbitMq.Opts.RoutingKey,
	}

	_, err = app.RabbitMq.Subscribe()

	if err != nil {
		log.Fatalf("Subscribe to rabbitmq consumer failed with error: %s", err.Error())
	}
}

func (app *NotifierApplication) InitLogger() {
	var err error

	app.Logger, err = zap.NewProduction()

	if err != nil {
		log.Fatalf("Application logger initialization failed with error: %s\n", err)
	}

	app.sugaredLogger = app.Logger.Sugar()
}

func (app *NotifierApplication) initConfig() {
	app.config = &Config{}

	if err := envconfig.Process("", app.config); err != nil {
		log.Fatalf("Config init failed with error: %s\n", err)
	}
}

func (app *NotifierApplication) Run() {
	if err := app.service.Run(); err != nil {
		log.Fatal(err)
	}
}

func (app *NotifierApplication) Process(ctx context.Context, o *proto.Order) error {
	h := handler.NewHandler(o, app.gRpcRepository, app.sugaredLogger, app.centrifugoClient, app.RabbitMq)

	if o.Status == constant.OrderStatusPaymentSystemDeclined || o.Status == constant.OrderStatusPaymentSystemCanceled {
		if err := h.SendCentrifugoMessage(o); err != nil {
			log.Println("[centrifugo]: " + err.Error())
		}
		return nil
	}

	n, err := h.GetNotifier()

	if err != nil {
		return err
	}

	n.Notify()

	if err := h.SendCentrifugoMessage(o); err != nil {
		log.Println("[centrifugo]: " + err.Error())
	}

	return nil
}
