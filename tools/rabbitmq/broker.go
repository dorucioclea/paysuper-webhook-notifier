package rabbitmq

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"log"
	"reflect"
)

type Broker struct {
	address  string
	rabbitMQ *rabbitMq

	subscriber *subscriber
}

func NewBroker(address string) (b *Broker) {
	b = &Broker{address: address}

	rmq := b.newRabbitMq()
	err := rmq.connect()

	if err != nil {
		log.Fatalf("[*] RabbitMq connection failed with error: %s", err)
	}

	b.rabbitMQ = rmq

	return
}

func (b *Broker) RegisterSubscriber(topic string, fn interface{}) error {
	if b.subscriber == nil {
		b.subscriber = initSubscriber(topic, b.rabbitMQ)
	}

	typ := reflect.TypeOf(fn)

	if typ.Kind() != reflect.Func {
		return errors.New("handler must have a function type")
	}

	tNum := typ.NumIn()

	if tNum != 2 {
		return errors.New("handler func must have two income argument")
	}

	if typ.In(1).Kind() != reflect.Map {
		return errors.New("second argument of handler func must have a amqp.Table type")
	}

	reqType := typ.In(0)
	isVal := false
	req := reflect.Value{}

	if reqType.Kind() == reflect.Ptr {
		req = reflect.New(reqType.Elem())
	} else {
		req = reflect.New(reqType)
		isVal = true
	}
	if isVal {
		req = req.Elem()
	}

	reqArg, ok := req.Interface().(proto.Message)

	if !ok {
		return errors.New("first argument of handler func must be instance of a proto.Message interface")
	}

	h := &handler{method: reflect.ValueOf(fn), reqArg: reqArg}
	b.subscriber.handlers = append(b.subscriber.handlers, h)

	return nil
}

func (b *Broker) Subscribe() error {
	return b.subscriber.Subscribe()
}
