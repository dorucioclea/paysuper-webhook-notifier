package internal

import (
	"context"
	"github.com/ProtocolONE/payone-notifier/internal/handler"
	"github.com/ProtocolONE/payone-repository/pkg/constant"
	proto "github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/ProtocolONE/payone-repository/pkg/proto/repository"
	"github.com/centrifugal/centrifuge"
	"github.com/micro/go-grpc"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
	"go.uber.org/zap"
	"log"
	"net/http"
)

type NotifierApplication struct {
	serviceContext context.Context
	serviceCancel  context.CancelFunc

	service        micro.Service
	gRpcRepository repository.RepositoryService
	sugaredLogger  *zap.SugaredLogger
	centrifugoNode *centrifuge.Node
	httpServer     *http.Server

	Logger *zap.Logger
}

func NewApplication() *NotifierApplication {
	return &NotifierApplication{}
}

func (app *NotifierApplication) Init() {
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
	app.initCentrifugo()
}

func (app *NotifierApplication) InitLogger() {
	var err error

	app.Logger, err = zap.NewProduction()

	if err != nil {
		log.Fatalf("Application logger initialization failed with error: %s\n", err)
	}

	app.sugaredLogger = app.Logger.Sugar()
}

func (app *NotifierApplication) initCentrifugo() {

}

func (app *NotifierApplication) Run() {
	//if err := app.service.Run(); err != nil {
	//	log.Fatal(err)
	//}

	if err := app.centrifugoNode.Run(); err != nil {
		log.Fatal(err)
	}

	//app.httpServer = &http.Server{Addr: ":8000"}
	//http.Handle("/websocket", centrifuge.NewWebsocketHandler(app.centrifugoNode, centrifuge.WebsocketConfig{}))

	//go func() {
	//	if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	//		log.Fatalf("listen: %s\n", err)
	//	}
	//}()
	http.Handle("/websocket", centrifuge.NewWebsocketHandler(app.centrifugoNode, centrifuge.WebsocketConfig{}))

	// Start HTTP server.
	//go func() {
		if err := http.ListenAndServe(":8000", nil); err != nil {
			panic(err)
		}
	//}()
}

func (app *NotifierApplication) Process(ctx context.Context, o *proto.Order) error {
	h, err := handler.NewHandler(o, app.gRpcRepository, app.sugaredLogger).GetNotifier()

	if err != nil {
		return err
	}

	h.Notify()

	return nil
}
