package main

import (
	"github.com/ProtocolONE/payone-notifier/internal"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/grpc"
	"log"
)

func main() {
	app := internal.NewApplication()
	app.Init()

	defer func() {
		if err := app.Logger.Sync(); err != nil {
			return
		}
	}()

	app.Run()

	log.Println("[x] The end")
}
