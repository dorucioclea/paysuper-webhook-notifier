package tools

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"math/rand"
)

func New(address string) {
	conn, err := amqp.Dial(address)

	if err != nil {
		log.Fatalf("Connection to RabbitMQ (address: %s) failed with error: %s", address, err.Error())
	}

	ch, err := conn.Channel()

	if err != nil {
		log.Fatalf("Opening RabbitMQ channel failed with error: %s", err.Error())
	}

	err = ch.ExchangeDeclare("test", "topic", true, false, false, false, nil)

	if err != nil {
		log.Fatal(err)
	}

	q, _ := ch.QueueDeclare(
		"test.queue",
		true,
		false,
		true,
		false,
		nil,
	)

	ch.QueueBind(q.Name, "*", "test", false, nil)

	ch1, _ := conn.Channel()

	err = ch1.ExchangeDeclare("test.timeout10", "topic", true, false, false, false, nil)

	if err != nil {
		log.Fatal(err)
	}

	q10, _ := ch1.QueueDeclare(
		"test.queue.timeout10",
		true,
		false,
		true,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "test",
			"x-message-ttl":             int32(1 * 1000),
			"x-dead-letter-routing-key": "*",
		},
	)

	ch.QueueBind(q.Name, "*", "test", false, nil)
	ch1.QueueBind(q10.Name, "*", "test.timeout10", false, nil)

	for i := 0; i < 10; i++ {
		ch.Publish(
			"test",
			fmt.Sprintf("msg%d", i),
			false,
			false,
			amqp.Publishing{ContentType: "text/plain", Body: []byte(fmt.Sprintf("msg %d", i))},
		)

		/*ch1.Publish(
			"test.timeout10",
			fmt.Sprintf("msg dlx %d", i),
			false,
			false,
			amqp.Publishing{ContentType: "text/plain", Body: []byte(fmt.Sprintf("msg dlx %d", i))},
		)*/
	}

	msg, _ := ch.Consume(q.Name, "123", true, false, false, false, nil)

	var rtr int32

	for {
		for d := range msg {
			rnd := rand.Intn(100)

			rtr1, ok := d.Headers["x-message-retry"]

			if !ok {
				rtr = 0
			} else {
				rtr = rtr1.(int32)+ 1
			}

			if rnd > 50 {
				ch1.Publish(
					"test.timeout10",
					d.RoutingKey,
					false,
					false,
					amqp.Publishing{
						ContentType: "text/plain",
						Body: d.Body,
						Headers: amqp.Table{"x-message-retry": rtr},
					})
				log.Printf("  [x] received message failde: %s", d.Body)
				continue
			}

			log.Printf("Retry count: %d", rtr)
			log.Printf("  [x] received message successfully complete: %s", d.Body)
		}
	}
}
