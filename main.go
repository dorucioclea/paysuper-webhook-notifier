package main

import (
	"github.com/ProtocolONE/payone-notifier/internal"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/grpc"
)

func main() {
	app := internal.NewApplication()
	app.Init()

	defer app.Stop()
	app.Run()
}