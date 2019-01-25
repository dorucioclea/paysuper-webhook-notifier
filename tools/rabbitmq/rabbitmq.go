package rabbitmq

import (
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"log"
	"sync"
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

type Opts map[string]bool

type rabbitMq struct {
	amqpUrl string

	conn    *amqp.Connection
	channel *amqp.Channel

	uuid string

	wg sync.WaitGroup
	sync.Mutex
	connected      bool
	close          chan bool
	waitConnection chan struct{}
}

func (b *Broker) newRabbitMq() (rmq *rabbitMq) {
	rmq = &rabbitMq{
		amqpUrl:        b.address,
		close:          make(chan bool),
		waitConnection: make(chan struct{}),
	}
	id, err := uuid.NewRandom()

	if err != nil {
		rmq.uuid = ""
	} else {
		rmq.uuid = id.String()
	}

	close(rmq.waitConnection)
	return
}

func (r *rabbitMq) DeclareExchange(name, kind string, opts Opts, args amqp.Table) error {
	pOpts := r.getOpts(opts, defaultExchangeOpts)

	return r.channel.ExchangeDeclare(
		name,
		kind,
		pOpts[optDurable],
		pOpts[optAutoDelete],
		pOpts[optInternal],
		pOpts[optNoWait],
		args,
	)
}

func (r *rabbitMq) DeclareQueue(name string, opts Opts, args amqp.Table) error {
	pOpts := r.getOpts(opts, defaultQueueOpts)

	_, err := r.channel.QueueDeclare(
		name,
		pOpts[optDurable],
		pOpts[optAutoDelete],
		pOpts[optExclusive],
		pOpts[optNoWait],
		args,
	)

	return err
}

func (r *rabbitMq) QueueBind(queue, key, exchange string, noWait bool, args amqp.Table) error {
	return r.channel.QueueBind(queue, key, exchange, noWait, args)
}

func (r *rabbitMq) Consume(queue string, opts Opts, args amqp.Table) (<-chan amqp.Delivery, error) {
	pOpts := r.getOpts(opts, defaultConsumeOpts)

	return r.channel.Consume(
		queue,
		r.uuid,
		pOpts[optAutoAck],
		pOpts[optExclusive],
		pOpts[optNoLocal],
		pOpts[optNoWait],
		args,
	)
}

func (r *rabbitMq) Publish(exchange, key string, opts Opts, message amqp.Publishing) error {
	pOpts := r.getOpts(opts, defaultPublishOpts)
	return r.channel.Publish(exchange, key, pOpts[optMandatory], pOpts[optImmediate], message)
}

func (r *rabbitMq) getOpts(rec, def Opts) Opts {
	if rec == nil {
		return def
	}

	proc := make(Opts)

	for k, v := range def {
		val, ok := rec[k]

		if !ok {
			val = v
		}

		proc[k] = val
	}

	return proc
}

func (r *rabbitMq) tryConnect() (err error) {
	if r.conn, err = amqp.Dial(r.amqpUrl); err != nil {
		return
	}

	log.Printf("[*] RabbitMQ connection created on address: %s", r.amqpUrl)

	if r.channel, err = r.conn.Channel(); err != nil {
		return
	}

	log.Println("[*] RabbitMQ channel created")

	return
}

func (r *rabbitMq) connect() (err error) {
	if err = r.tryConnect(); err != nil {
		return
	}

	r.Lock()
	r.connected = true
	r.Unlock()

	go r.reconnect()
	return
}

func (r *rabbitMq) reconnect() {
	var connect bool

	for {
		if connect {
			if err := r.tryConnect(); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			r.Lock()
			r.connected = true
			r.Unlock()

			close(r.waitConnection)
		}

		connect = true
		notifyClose := make(chan *amqp.Error)
		r.conn.NotifyClose(notifyClose)

		select {
		case <-notifyClose:
			r.Lock()
			r.connected = false
			r.waitConnection = make(chan struct{})
			r.Unlock()
		case <-r.close:
			return
		}
	}
}
