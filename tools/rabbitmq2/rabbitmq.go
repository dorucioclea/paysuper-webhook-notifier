package rabbitmq2

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"time"
)

const (
	optDurable    = "durable"
	optAutoDelete = "autoDelete"
	optExclusive  = "exclusive"
	optNoWait     = "noWait"
	optAutoAck    = "autoAck"
	optNoLocal    = "noLocal"
	optMandatory  = "mandatory"
	optImmediate  = "immediate"
	optInternal   = "internal"

	errorNicConnection = "connection not open"
	errorNilChannel    = "channel not open"
)

var defaultExchangeOpts = Opts{
	optDurable:    true,
	optAutoDelete: false,
	optInternal:   false,
	optNoWait:     false,
}

var defaultQueueOpts = Opts{
	optDurable:    true,
	optAutoDelete: false,
	optExclusive:  true,
	optNoWait:     false,
}

var defaultConsumeOpts = Opts{
	optAutoAck:   true,
	optExclusive: false,
	optNoLocal:   false,
	optNoWait:    false,
}

var defaultPublishOpts = Opts{
	optMandatory: false,
	optImmediate: false,
}

type Opts map[string]interface{}

type QueueOpts struct {
	Name string
	Opts Opts
	Args amqp.Table
}

type ExchangeOpts struct {
	Name string
	Kind string
	Opts Opts
	Args amqp.Table
}

type QueueBindOpts struct {
	Key    string
	NoWait bool
	Args   amqp.Table
}

type ConsumeOpts struct {
	Opts Opts
	Args amqp.Table
}

type rabbitMq struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	amqpUrl string
	done    chan error

	uuid           string
	prefetchCount  int
	prefetchSize   int
	prefetchGlobal bool
}

func (rmq *rabbitMq) Connect() (err error) {
	rmq.conn, err = amqp.Dial(rmq.amqpUrl)

	if err != nil {
		return fmt.Errorf("dial RabbitMQ server failed with error: %s", err)
	}

	log.Printf("[*] Connect to RabbitMQ server: %s", rmq.amqpUrl)

	go func() {
		// wait here for the channel to be closed
		log.Printf("[*] Closing RabbitMQ connection: %s", <-rmq.conn.NotifyClose(make(chan *amqp.Error)))
		// let handle know it's not time to reconnect
		rmq.done <- errors.New("channel closed")
	}()

	log.Printf("[*] Create RabbitMQ channel")

	rmq.channel, err = rmq.conn.Channel()

	if err != nil {
		return fmt.Errorf("creating RabbitMQ channel failed with error: %s", err)
	}

	log.Printf("[*] Set RabbitMQ channel QoS as: count %d; size: %d, global: %t", rmq.prefetchCount, rmq.prefetchSize, rmq.prefetchGlobal)

	err = rmq.channel.Qos(rmq.prefetchCount, rmq.prefetchSize, rmq.prefetchGlobal)

	if err != nil {
		return err
	}

	return
}

func (rmq *rabbitMq) reConnect(queueName, bindingKey string) (dlv <-chan amqp.Delivery, err error) {
	time.Sleep(30 * time.Second)

	if err = rmq.Connect(); err != nil {
		log.Printf("[*] ReConnectiong to RabbitMQ server failed with error: %s", err)
	}

	dlv, err = rmq.DeclareQueue(queueName, bindingKey)

	if err != nil {
		return deliveries, errors.New("Couldn't connect")
	}

	return deliveries, nil
}

func (rmq *rabbitMq) Consume(opts *SubscriberOpts) (dlv <-chan amqp.Delivery, err error) {
	log.Printf("[*] Declare RabbitMQ exchange %s", opts.ExchangeOpts.Name)

	err = rmq.channel.ExchangeDeclare(
		opts.ExchangeOpts.Name,
		opts.ExchangeOpts.Kind,
		opts.ExchangeOpts.Opts[optDurable].(bool),
		opts.ExchangeOpts.Opts[optAutoDelete].(bool),
		opts.ExchangeOpts.Opts[optInternal].(bool),
		opts.ExchangeOpts.Opts[optNoWait].(bool),
		opts.ExchangeOpts.Args,
	)

	if err != nil {
		return
	}

	log.Printf("[*] Declare RabbitMQ queue %s", opts.QueueOpts.Name)

	_, err = rmq.channel.QueueDeclare(
		opts.QueueOpts.Name,
		opts.QueueOpts.Opts[optDurable].(bool),
		opts.QueueOpts.Opts[optAutoDelete].(bool),
		opts.QueueOpts.Opts[optExclusive].(bool),
		opts.QueueOpts.Opts[optNoWait].(bool),
		opts.QueueOpts.Args,
	)

	if err != nil {
		return
	}

	log.Printf(
		"[*] Bind RabbitMQ queue %s to exchange %s with routing key %s",
		opts.QueueOpts.Name,
		opts.ExchangeOpts.Name,
		opts.QueueBindOpts.Key,
	)

	err = rmq.channel.QueueBind(
		opts.QueueOpts.Name,
		opts.QueueBindOpts.Key,
		opts.ExchangeOpts.Name,
		opts.QueueBindOpts.NoWait,
		opts.QueueBindOpts.Args,
	)

	if err != nil {
		return
	}

	consumer := opts.ExchangeOpts.Name + ":" + rmq.uuid

	log.Printf(
		"[*] Start RabbitMQ consumer %s for queue %s and exchange %s",
		consumer,
		opts.QueueOpts.Name,
		opts.ExchangeOpts.Name,
	)

	dlv, err = rmq.channel.Consume(
		opts.QueueOpts.Name,
		consumer,
		opts.ConsumeOpts.Opts[optAutoAck].(bool),
		opts.ConsumeOpts.Opts[optExclusive].(bool),
		opts.ConsumeOpts.Opts[optNoLocal].(bool),
		opts.ConsumeOpts.Opts[optNoWait].(bool),
		opts.ConsumeOpts.Args,
	)

	if err != nil {
		return
	}

	return
}
