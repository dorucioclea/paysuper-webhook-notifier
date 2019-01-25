package rabbitmq

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"reflect"
	"sync"
	"time"
)

const (
	defaultExchangeKind = "topic"
	defaultQueueBindKey = "*"
	defaultContentType  = "application/protobuf"
)

type QueueOpts struct {
	Name string
	Opts Opts
	Args amqp.Table
}

type ExchangeOpts struct {
	name string
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

type handler struct {
	method reflect.Value
	reqArg proto.Message
}

type subscriber struct {
	topic    string
	handlers []*handler
	fn       func(msg amqp.Delivery)

	mtx    sync.Mutex
	mayRun bool
	rabbit *rabbitMq

	Opts struct {
		*QueueOpts
		*ExchangeOpts
		*QueueBindOpts
		*ConsumeOpts
	}
}

func initSubscriber(topic string, rmq *rabbitMq) (subs *subscriber) {
	subs = &subscriber{
		topic:    topic,
		rabbit:   rmq,
		handlers: []*handler{},
	}

	subs.Opts.ExchangeOpts = &ExchangeOpts{
		name: topic,
		Kind: defaultExchangeKind,
		Opts: defaultExchangeOpts,
		Args: nil,
	}
	subs.Opts.QueueOpts = &QueueOpts{
		Name: topic + ".queue",
		Opts: defaultQueueOpts,
		Args: nil,
	}
	subs.Opts.QueueBindOpts = &QueueBindOpts{
		Key:    defaultQueueBindKey,
		NoWait: false,
		Args:   nil,
	}
	subs.Opts.ConsumeOpts = &ConsumeOpts{
		Opts: defaultConsumeOpts,
		Args: nil,
	}

	return
}

func (s *subscriber) Subscribe() (err error) {
	if s.rabbit.conn == nil {
		return errors.New(errorNicConnection)
	}

	if s.rabbit.channel == nil {
		return errors.New(errorNilChannel)
	}

	fn := func(msg amqp.Delivery) {
		if msg.ContentType != defaultContentType {
			if s.Opts.ConsumeOpts.Opts[optAutoAck] == false {
				_ = msg.Nack(false, false)
			}

			return
		}

		for _, h := range s.handlers {
			err = proto.Unmarshal(msg.Body, h.reqArg)

			if err != nil {
				continue
			}

			returnValues := h.method.Call([]reflect.Value{reflect.ValueOf(h.reqArg), reflect.ValueOf(msg.Headers)})

			if err := returnValues[0].Interface(); err != nil {
				if s.Opts.ConsumeOpts.Opts[optAutoAck] == false {
					_ = msg.Nack(false, false)
				}
			} else {
				if s.Opts.ConsumeOpts.Opts[optAutoAck] == false {
					_ = msg.Ack(false)
				}
			}

			return
		}
	}

	s.fn = fn
	s.mayRun = true

	go s.resubscribe()

	return
}

func (s *subscriber) resubscribe() {
	minResubscribeDelay := 100 * time.Millisecond
	maxResubscribeDelay := 30 * time.Second
	expFactor := time.Duration(2)
	reSubscribeDelay := minResubscribeDelay

	for {
		s.mtx.Lock()
		mayRun := s.mayRun
		s.mtx.Unlock()

		if !mayRun {
			return
		}

		select {
		case <-s.rabbit.close:
			return
		case <-s.rabbit.waitConnection:
		}

		s.rabbit.Lock()
		if !s.rabbit.connected {
			s.rabbit.Unlock()
			continue
		}

		sub, err := s.consume()
		s.rabbit.Unlock()

		switch err {
		case nil:
			reSubscribeDelay = minResubscribeDelay
		default:
			if reSubscribeDelay > maxResubscribeDelay {
				reSubscribeDelay = maxResubscribeDelay
			}
			time.Sleep(reSubscribeDelay)
			reSubscribeDelay *= expFactor
			continue
		}

		for d := range sub {
			s.rabbit.wg.Add(1)

			go func(d amqp.Delivery) {
				s.fn(d)
				s.rabbit.wg.Done()
			}(d)
		}
	}
}

func (s *subscriber) consume() (dls <-chan amqp.Delivery, err error) {
	err = s.rabbit.DeclareQueue(s.Opts.QueueOpts.Name, s.Opts.QueueOpts.Opts, s.Opts.QueueOpts.Args)

	if err != nil {
		return
	}

	err = s.rabbit.QueueBind(
		s.Opts.QueueOpts.Name,
		s.Opts.QueueBindOpts.Key,
		s.Opts.ExchangeOpts.name,
		s.Opts.QueueBindOpts.NoWait,
		s.Opts.QueueBindOpts.Args,
	)

	if err != nil {
		return
	}

	dls, err = s.rabbit.Consume(s.Opts.QueueOpts.Name, s.Opts.ConsumeOpts.Opts, s.Opts.ConsumeOpts.Args)

	if err != nil {
		return
	}

	return
}
