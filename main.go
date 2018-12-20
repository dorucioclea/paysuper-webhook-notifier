package main

import (
	"github.com/ProtocolONE/payone-notifier/internal"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/transport/grpc"
)

func main() {
	app := internal.NewApplication()
	app.Init()
	app.InitLogger()

	defer func() {
		if err := app.Logger.Sync(); err != nil {
			return
		}
	}()

	app.Run()
}
