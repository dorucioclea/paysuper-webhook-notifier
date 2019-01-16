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
	log.Printf("[Notifier_DEBUG] start notification for order id: %s", n.order.Id)

	n.order.Status = constant.OrderStatusProjectComplete

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		log.Printf("[Notifier_DEBUG] notification for order id failed: %s", n.order.Id)
	}
}
