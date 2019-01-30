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

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.logger.Error("[PAYONE_NOTIFIER] update order failed", err, n.order.Id, notifierHandlerEmpty)
	}
}
