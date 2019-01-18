package main

import (
	"github.com/ProtocolONE/payone-notifier/internal"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/grpc"
)

func main() {
	app := internal.NewApplication()
	app.InitLogger()
	app.Init()

	defer func() {
		if err := app.Logger.Sync(); err != nil {
			return
		}
	}()

	defer func() {
		if err := app.RmqConn.Close(); err != nil {
			return
		}
	}()

	defer func() {
		if err := app.RmqChan.Close(); err != nil {
			return
		}
	}()

	app.Run()
}
