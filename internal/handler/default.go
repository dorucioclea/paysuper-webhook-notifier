package handler

import (
	"context"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
)

type Default Empty

func newDefaultHandler(h *Handler) Notifier {
	return &Default{Handler: h}
}

func (n *Default) Notify() error {
	n.order.Status = constant.OrderStatusProjectComplete

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}
