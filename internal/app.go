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
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
	k8s "github.com/micro/kubernetes/go/micro"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"log"
	"net/http"
)

const (
	dlxQueueName    = "notifier-queue-dlx"
	dlxExchangeName = "notifier-exchange-dlx"
	dlxDelay        = 10
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

	RmqConn *amqp.Connection
	RmqChan *amqp.Channel

	Logger *zap.Logger
}

func NewApplication() *NotifierApplication {
	return &NotifierApplication{}
}

func (app *NotifierApplication) Init() {
	app.initConfig()

	options := []micro.Option{
		micro.Name(constant.PayOneSubscriberNotifierName),
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

	//app.initRmqDlxPublisher()
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

func (app *NotifierApplication) initRmqDlxPublisher() {
	rmqConn, err := amqp.Dial(app.config.BrokerAddress)
	if err != nil {
		panic(err)
	}
	defer rmqConn.Close()

	rmqChan, err := rmqConn.Channel()
	if err != nil {
		panic(err)
	}
	defer rmqChan.Close()

	args := make(amqp.Table)
	args["x-dead-letter-exchange"] = "errors"
	args["x-dead-letter-routing-key"] = "listening"

	listen, err := rmqChan.QueueDeclare("listening", true, false, false, false, args)
	if err != nil {
		panic(err)
	}

	args["x-dead-letter-routing-key"] = "publish"
	publish, err := rmqChan.QueueDeclare("publish", true, false, false, false, args)
	if err != nil {
		panic(err)
	}

	msg, err := rmqChan.Consume(listen.Name, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	rmqChan.Publish("", publish.Name, false, false, amqp.Publishing{Body: []byte("123")})

	forever := make(chan bool)

	go func() {
		for d := range msg {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (app *NotifierApplication) Run() {
	if err := app.service.Run(); err != nil {
		log.Fatal(err)
	}
}

func (app *NotifierApplication) Process(ctx context.Context, o *proto.Order) error {
	h := handler.NewHandler(o, app.gRpcRepository, app.sugaredLogger, app.centrifugoClient)

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
