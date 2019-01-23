package handler

import (
	"context"
	"github.com/ProtocolONE/payone-repository/pkg/constant"
	"log"
)

type Empty struct {
	*Handler
}

func newEmptyHandler(h *Handler) Notifier {
	return &Empty{Handler: h}
}

func (n *Empty) Notify() {
	n.order.Status = constant.OrderStatusProjectComplete

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		log.Printf("[Notifier_DEBUG] notification for order id failed with error %s", err.Error())
	}

	log.Println("Empty notify done")

	if err := n.rabbitMq.Publish("123"); err != nil {
		log.Println("Publish error" + err.Error())
	}
}
