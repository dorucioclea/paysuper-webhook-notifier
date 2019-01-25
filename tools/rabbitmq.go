package tools

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"log"
)

const (
	defaultRoutingKey = "notifier"
)

type ExchangeOpts struct {
	Name        string
	Type        string
	Durable     bool
	AutoDeleted bool
	Internal    bool
	NoWait      bool
	Args        amqp.Table
}

type QueueOpts struct {
	Name        string
	Durable     bool
	AutoDeleted bool
	Exclusive   bool
	NoWait      bool
	Args        amqp.Table
}

type PublishOpts struct {
	Mandatory bool
	Immediate bool
}

type QueueBindOpts struct {
	NoWait bool
	Args   amqp.Table
}

type ConsumeOpts struct {
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}

type Options struct {
	Exchange   *ExchangeOpts
	Queue      *QueueOpts
	Publish    *PublishOpts
	QueueBind  *QueueBindOpts
	Consume    *ConsumeOpts
	RoutingKey string
}

type RabbitMq struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	Opts    *Options
}

func NewRabbitMq(address string) *RabbitMq {
	conn, err := amqp.Dial(address)

	if err != nil {
		log.Fatalf("Connection to RabbitMQ (address: %s) failed with error: %s", address, err.Error())
	}

	ch, err := conn.Channel()

	if err != nil {
		log.Fatalf("Opening RabbitMQ channel failed with error: %s", err.Error())
	}

	inst := &RabbitMq{conn: conn, channel: ch}
	inst.Opts = &Options{
		Exchange:   inst.getDefaultExchangeOpts(),
		Queue:      inst.getDefaultQueueOpts(),
		Publish:    inst.getDefaultPublishOpts(),
		QueueBind:  inst.getDefaultQueueBindOpts(),
		Consume:    inst.getDefaultConsumeOpts(),
		RoutingKey: defaultRoutingKey,
	}

	return inst
}

func (rmq *RabbitMq) Publish(msg interface{}) error {
	err := rmq.channel.ExchangeDeclare(
		rmq.Opts.Exchange.Name,
		rmq.Opts.Exchange.Type,
		rmq.Opts.Exchange.Durable,
		rmq.Opts.Exchange.AutoDeleted,
		rmq.Opts.Exchange.Internal,
		rmq.Opts.Exchange.NoWait,
		rmq.Opts.Exchange.Args,
	)

	if err != nil {
		return err
	}

	b, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	err = rmq.channel.Publish(
		rmq.Opts.Exchange.Name,
		rmq.Opts.RoutingKey,
		rmq.Opts.Publish.Mandatory,
		rmq.Opts.Publish.Immediate,
		amqp.Publishing{ContentType: "application/json", Body: b},
	)

	if err != nil {
		return err
	}

	log.Printf("[x] Sent message to queue %s", string(b))

	return nil
}

func (rmq *RabbitMq) Subscribe() (<-chan string, error) {
	out := make(chan string)

	err := rmq.channel.ExchangeDeclare(
		rmq.Opts.Exchange.Name,
		rmq.Opts.Exchange.Type,
		rmq.Opts.Exchange.Durable,
		rmq.Opts.Exchange.AutoDeleted,
		rmq.Opts.Exchange.Internal,
		rmq.Opts.Exchange.NoWait,
		rmq.Opts.Exchange.Args,
	)

	if err != nil {
		return nil, err
	}

	q, err := rmq.channel.QueueDeclare(
		rmq.Opts.Queue.Name,
		rmq.Opts.Queue.Durable,
		rmq.Opts.Queue.AutoDeleted,
		rmq.Opts.Queue.Exclusive,
		rmq.Opts.Queue.NoWait,
		rmq.Opts.Queue.Args,
	)

	if err != nil {
		return nil, err
	}

	err = rmq.channel.QueueBind(
		q.Name,
		rmq.Opts.RoutingKey,
		rmq.Opts.Exchange.Name,
		rmq.Opts.QueueBind.NoWait,
		rmq.Opts.QueueBind.Args,
	)

	if err != nil {
		return nil, err
	}

	msg, err := rmq.channel.Consume(
		q.Name,
		rmq.Opts.Consume.Consumer,
		rmq.Opts.Consume.AutoAck,
		rmq.Opts.Consume.Exclusive,
		rmq.Opts.Consume.NoLocal,
		rmq.Opts.Consume.NoWait,
		rmq.Opts.Consume.Args,
	)

	if err != nil {
		return nil, err
	}

	go func() {
		for d := range msg {
			log.Printf("  [x] received message: %s", d.Body)
			out <- string(d.Body)
		}
	}()

	log.Printf("[x] notifier consumer %s ready to receive", rmq.Opts.Consume.Consumer)

	return out, nil
}

func (rmq *RabbitMq) Stop() {
	func() {
		if err := rmq.conn.Close(); err != nil {
			log.Printf("[x] RabbitMq connection close failed with error: %s", err.Error())
		}
	}()

	func() {
		if err := rmq.channel.Close(); err != nil {
			log.Printf("[x] RabbitMq channel close failed with error: %s", err.Error())
		}
	}()
}

func (rmq *RabbitMq) getDefaultExchangeOpts() *ExchangeOpts {
	opts := &ExchangeOpts{
		Name:        "notifier.exchange",
		Type:        "topic",
		Durable:     true,
		AutoDeleted: false,
		Internal:    false,
		NoWait:      false,
		Args:        nil,
	}

	return opts
}

func (rmq *RabbitMq) getDefaultQueueOpts() *QueueOpts {
	opts := &QueueOpts{
		Name:        "notifier",
		Durable:     false,
		AutoDeleted: false,
		Exclusive:   true,
		NoWait:      false,
		Args:        nil,
	}

	return opts
}

func (rmq *RabbitMq) getDefaultPublishOpts() *PublishOpts {
	return &PublishOpts{Mandatory: false, Immediate: false}
}

func (rmq *RabbitMq) getDefaultQueueBindOpts() *QueueBindOpts {
	return &QueueBindOpts{NoWait: false, Args: nil}
}

func (rmq *RabbitMq) getDefaultConsumeOpts() *ConsumeOpts {
	id, _ := uuid.NewRandom()

	opts := &ConsumeOpts{
		Consumer:  id.String(),
		AutoAck:   true,
		Exclusive: false,
		NoLocal:   false,
		NoWait:    false,
		Args:      nil,
	}

	return opts
}
