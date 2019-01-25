package rabbitmq2

import (
	"github.com/google/uuid"
	"math/rand"
	"strconv"
)

type Broker struct {
	rabbitMq   *rabbitMq
	subscriber *subscriber
	publisher  *publisher
}

type BrokerOpts struct {
	PrefetchCount  int
	PrefetchSize   int
	PrefetchGlobal bool
}

func NewBroker(addr string, opts *BrokerOpts) (br *Broker) {
	br = &Broker{
		rabbitMq: &rabbitMq{
			amqpUrl: addr,
			done:    make(chan error),
		},
		subscriber: &subscriber{
			handlers: []*handler{},
		},
		publisher: &publisher{},
	}

	if opts != nil {
		br.rabbitMq.prefetchCount = opts.PrefetchCount
		br.rabbitMq.prefetchSize = opts.PrefetchSize
		br.rabbitMq.prefetchGlobal = opts.PrefetchGlobal
	}

	rmqUuid, err := uuid.NewRandom()

	if err != nil {
		br.rabbitMq.uuid = strconv.Itoa(rand.Intn(9999999999))
	} else {
		br.rabbitMq.uuid = rmqUuid.String()
	}

	return
}

func (br *Broker) RegisterHandler(fn interface{}) error {
	return br.subscriber.registerHandler(fn)
}

func (br *Broker) Subscribe(opts *SubscriberOpts) {
	br.subscriber.initSubscriberOpts()

}

func (br *Broker) Publish() {

}
