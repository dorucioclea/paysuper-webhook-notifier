package handler

import (
	"context"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"go.uber.org/zap"
)

type Empty struct {
	*Handler
}

func newEmptyHandler(h *Handler) Notifier {
	return &Empty{Handler: h}
}

func (n *Empty) Notify() error {
	if n.order.PrivateStatus != constant.OrderStatusPaymentSystemComplete {
		return nil
	}

	n.order.PrivateStatus = constant.OrderStatusProjectComplete
	_, err := n.repository.UpdateOrder(context.TODO(), n.order)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationUpdate, err, nil)
	} else {
		zap.L().Info("order status successfully changed", zap.String("id", n.order.Id))
	}

	return nil
}
