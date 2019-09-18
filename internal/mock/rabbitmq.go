package mock

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/streadway/amqp"
	rabbitmq "gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
)

type BrokerMockOk struct{}
type BrokerMockError struct{}

func NewBrokerMockOk() rabbitmq.BrokerInterface {
	return &BrokerMockOk{}
}

func NewBrokerMockError() rabbitmq.BrokerInterface {
	return &BrokerMockError{}
}

func (b *BrokerMockOk) RegisterSubscriber(topic string, fn interface{}) error {
	return nil
}

func (b *BrokerMockOk) Subscribe(exit chan bool) error {
	return nil
}

func (b *BrokerMockOk) Publish(topic string, msg proto.Message, h amqp.Table) error {
	return nil
}

func (b *BrokerMockOk) SetExchangeName(name string) {
	return
}

func (b *BrokerMockError) RegisterSubscriber(topic string, fn interface{}) error {
	return errors.New("some error")
}

func (b *BrokerMockError) Subscribe(exit chan bool) error {
	return errors.New("some error")
}

func (b *BrokerMockError) Publish(topic string, msg proto.Message, h amqp.Table) error {
	return nil
}

func (b *BrokerMockError) SetExchangeName(name string) {
	return
}
