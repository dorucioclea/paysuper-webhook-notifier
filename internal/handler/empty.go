package handler

import (
	"context"
	"github.com/ProtocolONE/payone-repository/pkg/constant"
)

type Empty struct {
	*Handler
}

func newEmptyHandler(h *Handler) Notifier {
	return &Empty{Handler: h}
}

func (n *Empty) Notify() {
	n.order.Status = constant.OrderStatusProjectComplete
	n.repository.UpdateOrder(context.TODO(), n.order)
}
