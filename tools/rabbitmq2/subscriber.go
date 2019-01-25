package rabbitmq2

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/streadway/amqp"
	"reflect"
)

type handler struct {
	method reflect.Value
	reqArg proto.Message
}

type subscriber struct {
	handlers []*handler
	opts     *SubscriberOpts
}

type SubscriberOpts struct {
	*QueueOpts
	*ExchangeOpts
	*QueueBindOpts
	*ConsumeOpts
}

func (s *subscriber) registerHandler(fn interface{}) (err error) {
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
	s.handlers = append(s.handlers, h)

	return
}

func (s *subscriber) initSubscriberOpts() {

}

func (s *subscriber) Consume(d <-chan amqp.Delivery) {

}
