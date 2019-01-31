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

func (n *Empty) Notify() error {
	n.order.Status = constant.OrderStatusProjectComplete

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}
