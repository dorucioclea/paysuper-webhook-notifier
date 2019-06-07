package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
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
)

var orderPublicStatusToEventNameMapping = map[string]string{
	constant.OrderPublicStatusProcessed:  eventNameSuccess,
	constant.OrderPublicStatusChargeback: eventNameChargeback,
	constant.OrderPublicStatusCanceled:   eventNameCancel,
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

	if order.Project.Status == pkg.ProjectStatusDeleted {
		if err := n.SendCentrifugoMessage(n.order, centrifugoMsgNotificationForDeletedProject); err != nil {
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
		n.HandleError(loggerErrorNotificationMalfored, err, nil)
		return errors.New(loggerErrorNotificationMalfored)
	}

	url := n.getNotificationUrl(ps)
	if url == "" {
		if err := n.SendCentrifugoMessage(order, centrifugoMsgNotificationUrlEmpty); err != nil {
			n.HandleError(LoggerNotificationCentrifugo, err, nil)
		}
		return errors.New(loggerErrorProjectUrlEmpty)
	}

	resp, sendErr := n.sendRequest(url, req, NotificationActionPayment)

	if sendErr != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		order.PrivateStatus = constant.OrderStatusProjectComplete
	} else {
		order.PrivateStatus = constant.OrderStatusProjectReject
	}

	err = n.setStat(statKey, ps, true)
	if err != nil {
		n.HandleError(LoggerNotificationRedis, err, nil)
	}

	order.SetNotificationStatus(ps, true)
	if _, err := n.repository.UpdateOrder(context.TODO(), order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}

func (n *Default) sendRequest(url string, req interface{}, action string) (*http.Response, error) {
	reqUrl, err := n.validateUrl(url)

	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		HeaderContentType:   MIMEApplicationJSON,
		HeaderAccept:        MIMEApplicationJSON,
		HeaderAuthorization: "Signature " + n.getSignature(b),
	}

	resp, err := n.request(http.MethodPost, reqUrl.String(), b, headers)

	if err != nil {
		return nil, err
	}

	oId := n.order.GetId()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent &&
		resp.StatusCode != http.StatusUnprocessableEntity {
		return nil, errors.New(fmt.Sprintf(errorNotificationNeedRetry, oId, action))
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

func (n *Default) getSignature(req []byte) string {
	h := sha256.New()
	h.Write([]byte(string(req) + n.order.GetProject().GetSecretKey()))

	return hex.EncodeToString(h.Sum(nil))
}

func (n *Default) getNotificationUrl(publicStatus string) string {
	switch publicStatus {
	case constant.OrderPublicStatusProcessed:
		return n.order.Project.UrlProcessPayment
	case constant.OrderPublicStatusChargeback:
		return n.order.Project.UrlChargebackPayment
	case constant.OrderPublicStatusCanceled:
		return n.order.Project.UrlCancelPayment
	case constant.OrderPublicStatusRefunded:
		return n.order.Project.UrlRefundPayment
	default:
		return ""
	}
}

func (n *Default) getNotificationEventName(publicStatus string) string {
	en, ok := orderPublicStatusToEventNameMapping[publicStatus]
	if ok {
		return en
	}
	return ""
}
