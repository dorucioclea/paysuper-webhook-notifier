package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	errorNoEventForCurrentStatus = "no event name for current order status"
	loggerErrorDeletedProject    = "project is deleted"
	loggerErrorProjectUrlEmpty   = "project url empty"

	eventNameSuccess    = "payment.success"
	eventNameChargeback = "payment.chargeback"
	eventNameCancel     = "payment.cancel"
	eventNameRefund     = "payment.refund"

	centrifugoMsgNotificationUrlEmpty          = "notification url is empty"
	centrifugoMsgNotificationForDeletedProject = "notification for deleted project"

	psNotificationsKeyMask = "ps:notify:%s"

	errorCantNotifyBillingServer = "can't notify billing server testing results"
	errorCantNotifyMerchantServer = "can't notify merchant in centrifugo"
)

var orderPublicStatusToEventNameMapping = map[string]string{
	constant.OrderPublicStatusProcessed:  eventNameSuccess,
	constant.OrderPublicStatusChargeback: eventNameChargeback,
	constant.OrderPublicStatusCanceled:   eventNameCancel,
	constant.OrderPublicStatusRejected:   eventNameCancel,
	constant.OrderPublicStatusRefunded:   eventNameRefund,
}

type Default Empty

type OrderNotificationMessage struct {
	Id          string         `json:"id"`
	Type        string         `json:"type"`
	Event       string         `json:"event"`
	Live        bool           `json:"live"`
	CreatedAt   string         `json:"created_at"`
	ExpiresAt   string         `json:"expires_at"`
	DeliveryTry int32          `json:"delivery_try"`
	Object      *billing.Order `json:"object"`
}

func newDefaultHandler(h *Handler) Notifier {
	return &Default{Handler: h}
}

func (n *Default) Notify() error {
	order := n.order
	notifyRequest := &grpc.NotifyWebhookTestResultsRequest{TestCase: order.TestingCase, ProjectId: order.GetProjectId(), Type: order.ProductType}

	if order.Project.Status == pkg.ProjectStatusDeleted {
		if err := n.sendToAdminCentrifugo(n.order, centrifugoMsgNotificationForDeletedProject); err != nil {
			n.HandleError(LoggerNotificationCentrifugo, err, nil)
		}
		return errors.New(loggerErrorDeletedProject)
	}

	statKey := fmt.Sprintf(psNotificationsKeyMask, order.Id)
	stat, err := n.getStat(statKey)
	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	ps := order.GetPublicStatus()

	// don't send notification for current status if it already sent
	if stat.Get(ps) == true {
		order.SetNotificationStatus(ps, true)
		if _, err := n.repository.UpdateOrder(context.TODO(), order); err != nil {
			n.HandleError(loggerErrorNotificationUpdate, err, nil)
		}
		return nil
	}

	req, err := n.getPaymentNotification()
	if err != nil {
		if len(order.TestingCase) == 0 {
			n.HandleError(loggerErrorNotificationMalformed, err, nil)
		}
		return errors.New(loggerErrorNotificationMalformed)
	}

	url := n.getNotificationUrl(ps)
	if url == "" {
		if len(order.TestingCase) > 0 {
			notifyRequest.IsPassed = false
			if _, err := n.repository.NotifyWebhookTestResults(context.TODO(), notifyRequest); err != nil {
				zap.S().Errorw(errorCantNotifyBillingServer, "err", err)
			}
			if err := n.sendToMerchantTestingCentrifugo(order, order.TestingCase, nil); err != nil {
				zap.S().Errorw(errorCantNotifyMerchantServer, "err", err)
			}
		} else {
			if err := n.sendToAdminCentrifugo(order, centrifugoMsgNotificationUrlEmpty); err != nil {
				n.HandleError(LoggerNotificationCentrifugo, err, nil)
			}
		}
		return errors.New(loggerErrorProjectUrlEmpty)
	}

	resp, sendErr := n.sendRequest(url, req, NotificationActionPayment)

	if sendErr != nil {
		if len(order.TestingCase) > 0 {
			notifyRequest.IsPassed = false
			if _, err := n.repository.NotifyWebhookTestResults(context.TODO(), notifyRequest); err != nil {
				zap.S().Errorw(errorCantNotifyBillingServer, "err", err)
			}
			if err := n.sendToMerchantTestingCentrifugo(order, order.TestingCase, nil); err != nil {
				zap.S().Errorw(errorCantNotifyMerchantServer, "err", err)
			}
		} else {
			return n.handleErrorWithRetry(loggerErrorNotificationRetry, sendErr, nil)
		}
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		if n.order.PrivateStatus == constant.OrderStatusPaymentSystemComplete {
			order.PrivateStatus = constant.OrderStatusProjectComplete
		}
		if len(order.TestingCase) != 0 {
			notifyRequest.IsPassed = order.TestingCase == pkg.TestCaseCorrectPayment || order.TestingCase == pkg.TestCaseExistingUser
			if _, err := n.repository.NotifyWebhookTestResults(context.TODO(), notifyRequest); err != nil {
				zap.S().Errorw(errorCantNotifyBillingServer, "err", err)
			}
			if err := n.sendToMerchantTestingCentrifugo(order, order.TestingCase, resp); err != nil {
				zap.S().Errorw(errorCantNotifyMerchantServer, "err", err)
			}
		}
	} else {
		order.PrivateStatus = constant.OrderStatusProjectReject
		if len(order.TestingCase) != 0 {
			notifyRequest.IsPassed = order.TestingCase == pkg.TestCaseIncorrectPayment || order.TestingCase == pkg.TestCaseNonExistingUser
			if _, err := n.repository.NotifyWebhookTestResults(context.TODO(), notifyRequest); err != nil {
				zap.S().Errorw(errorCantNotifyBillingServer, "err", err)
			}
			if err := n.sendToMerchantTestingCentrifugo(order, order.TestingCase, resp); err != nil {
				zap.S().Errorw(errorCantNotifyMerchantServer, "err", err)
			}
		}
	}

	// We don't want update virtual orders
	if len(order.TestingCase) == 0 {
		err = n.setStat(statKey, ps, true)
		if err != nil {
			n.HandleError(LoggerNotificationRedis, err, nil)
		}

		order.SetNotificationStatus(ps, true)
		if _, err := n.repository.UpdateOrder(context.TODO(), order); err != nil {
			n.HandleError(loggerErrorNotificationUpdate, err, nil)
		}
	}

	return nil
}

func (n *Default) sendRequest(url string, req interface{}, action string) (*http.Response, error) {
	resp, err := n.sender.Send(url, req, action, n.order.GetProject().GetSecretKey())
	if err != nil {
		if err.Error() == errorHttpRequestFailed {
			return nil, errors.New(fmt.Sprintf(errorNotificationNeedRetry, n.order.GetId(), action))
		}
		return nil, err
	}

	return resp, nil
}

func (n *Default) getPaymentNotification() (*OrderNotificationMessage, error) {

	ps := n.order.GetPublicStatus()

	event := n.getNotificationEventName(ps)
	if event == "" {
		return nil, errors.New(errorNoEventForCurrentStatus)
	}

	h := sha256.New()
	h.Write([]byte(n.order.Id + event))

	res := &OrderNotificationMessage{
		Id:          hex.EncodeToString(h.Sum(nil)),
		Type:        "notification",
		Event:       event,
		CreatedAt:   time.Now().Format(time.RFC3339),
		DeliveryTry: n.RetryCount,
		Object:      n.order,
	}

	res.Live = n.order.Project.Status == pkg.ProjectStatusInProduction

	return res, nil
}

func (n *Default) getNotificationUrl(publicStatus string) string {
	//INFO According #192488 we need to use just one webhook URL for all kind of notifications.
	return n.order.Project.UrlProcessPayment
}

func (n *Default) getNotificationEventName(publicStatus string) string {
	en, ok := orderPublicStatusToEventNameMapping[publicStatus]
	if ok {
		return en
	}
	return ""
}