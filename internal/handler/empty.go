package handler

import (
	"context"
	"github.com/paysuper/paysuper-proto/go/recurringpb"
)

type Empty struct {
	*Handler
}

func newEmptyHandler(h *Handler) Notifier {
	return &Empty{Handler: h}
}

func (n *Empty) Notify() error {
	if n.order.PrivateStatus != recurringpb.OrderStatusPaymentSystemComplete {
		return nil
	}

	n.order.PrivateStatus = recurringpb.OrderStatusProjectComplete
	_, err := n.repository.UpdateOrder(context.TODO(), n.order)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}
