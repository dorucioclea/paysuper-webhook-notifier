package main

import (
	"fmt"
	"github.com/ProtocolONE/payone-notifier/tools/rabbitmq"
	"github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/gogo/protobuf/proto"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/grpc"
	"github.com/streadway/amqp"
	"log"
)

func main() {
	/*app := internal.NewApplication()
	app.InitLogger()
	app.Init()

	defer func() {
		if err := app.Logger.Sync(); err != nil {
			return
		}
	}()

	app.Run()*/

	//tools.New("amqp://127.0.0.1:5672")

	s := &st{}

	br := rabbitmq.NewBroker("amqp://127.0.0.1:5672")

	if err := br.RegisterSubscriber("test3", s.test); err != nil {
		log.Fatalln(err)
	}

	if err := br.RegisterSubscriber("test3", s.test2); err != nil {
		log.Fatalln(err)
	}

	for i := 0; i < 10; i++ {
		var pbf []byte
		var err error

		st := &billing.Name{
			En: fmt.Sprintf("test %d", i),
			Ru: fmt.Sprintf("тест %d", i),
		}
		pbf, err = proto.Marshal(st)

		if err != nil {
			log.Printf("Message marshaling is failed. Message id: %d ===> %s\n", i, err.Error())
			continue
		}

		err = br.Publish("test3", amqp.Publishing{ContentType: "application/protobuf", Body: pbf})

		if err != nil {
			log.Printf("Message publishing failed. Message id: %d ===> %s\n", i, err.Error())
			continue
		}

		log.Printf("Message publish success: %s\n", st.En)
	}

	if err := br.Subscribe(); err != nil {
		log.Fatalln(err)
	}

	log.Println("test")
}

type st struct {}

func (s *st) test(a *billing.Name, b amqp.Table) (headers amqp.Table, err error) {
	log.Println("Processed message method test: " + a.En)

	return
}

func (s *st) test2(a *billing.Country, b amqp.Table) (headers amqp.Table, err error) {
	log.Println("Processed message method test2: " + a.Name.En)

	return
}
