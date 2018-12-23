package internal

import (
	"context"
	"github.com/ProtocolONE/payone-notifier/internal/handler"
	"github.com/ProtocolONE/payone-repository/pkg/constant"
	proto "github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/ProtocolONE/payone-repository/pkg/proto/repository"
	"github.com/ProtocolONE/payone-repository/tools"
	"github.com/centrifugal/gocent"
	"github.com/kelseyhightower/envconfig"
	"github.com/micro/go-grpc"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
	"go.uber.org/zap"
	"log"
	"net/http"
)

type Config struct {
	CentrifugoUrl string `envconfig:"CENTRIFUGO_URL" required:"true"`
	CentrifugoKey string `envconfig:"CENTRIFUGO_KEY" required:"true"`
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

	Logger *zap.Logger
}

func NewApplication() *NotifierApplication {
	return &NotifierApplication{}
}

func (app *NotifierApplication) Init() {
	app.initConfig()

	app.service = micro.NewService(
		micro.Name(constant.PayOneSubscriberNotifierName),
		micro.Version(constant.PayOneMicroserviceVersion),
	)
	app.service.Init()

	err := micro.RegisterSubscriber(constant.PayOneTopicNotifyPaymentName, app.service.Server(), app.Process, server.SubscriberQueue("queue.pubsub"))

	if err != nil {
		log.Fatal(err)
	}

	service := grpc.NewService(
		micro.Name(constant.PayOneRepositoryServiceName),
		micro.Context(app.serviceContext),
	)
	service.Init()

	app.gRpcRepository = repository.NewRepositoryService(constant.PayOneRepositoryServiceName, service.Client())
	app.centrifugoClient = gocent.New(
		gocent.Config{
			Addr:       app.config.CentrifugoUrl,
			Key:        app.config.CentrifugoKey,
			HTTPClient: tools.NewLoggedHttpClient(app.sugaredLogger),
		},
	)
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
	h := handler.NewHandler(o, app.gRpcRepository, app.sugaredLogger, app.centrifugoClient)

	if o.Status == constant.OrderStatusPaymentSystemDeclined || o.Status == constant.OrderStatusPaymentSystemCanceled {
		return h.SendCentrifugoMessage(o)
	}

	n, err := h.GetNotifier()

	if err != nil {
		return err
	}

	n.Notify()

	return h.SendCentrifugoMessage(o)
}
