package handler

import (
	"context"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
)

type Empty struct {
	*Handler
}

func newEmptyHandler(h *Handler) Notifier {
	return &Empty{Handler: h}
}

func (n *Empty) Notify() error {
	n.order.PrivateStatus = constant.OrderStatusProjectComplete
	_, err := n.repository.UpdateOrder(context.TODO(), n.order)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}
