package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/paysuper/paysuper-proto/go/billingpb"
	"github.com/paysuper/paysuper-proto/go/recurringpb"
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

	errorNotSuccessStatus = "status is not success"
)

var orderPublicStatusToEventNameMapping = map[string]string{
	recurringpb.OrderPublicStatusProcessed:  eventNameSuccess,
	recurringpb.OrderPublicStatusChargeback: eventNameChargeback,
	recurringpb.OrderPublicStatusCanceled:   eventNameCancel,
	recurringpb.OrderPublicStatusRejected:   eventNameCancel,
	recurringpb.OrderPublicStatusRefunded:   eventNameRefund,
}

type Default Empty

type OrderNotificationMessage struct {
	Id          string           `json:"id"`
	Type        string           `json:"type"`
	Event       string           `json:"event"`
	Live        bool             `json:"live"`
	CreatedAt   string           `json:"created_at"`
	ExpiresAt   string           `json:"expires_at"`
	DeliveryTry int32            `json:"delivery_try"`
	Object      *billingpb.Order `json:"object"`
}

func newDefaultHandler(h *Handler) Notifier {
	return &Default{Handler: h}
}

func (n *Default) Notify() error {
	order := n.order

	if order.Project.Status == billingpb.ProjectStatusDeleted {
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
		n.HandleError(loggerErrorNotificationMalformed, err, nil)
		return errors.New(loggerErrorNotificationMalformed)
	}

	url := n.getNotificationUrl(ps)
	if url == "" {
		if err := n.sendToAdminCentrifugo(order, centrifugoMsgNotificationUrlEmpty); err != nil {
			n.HandleError(LoggerNotificationCentrifugo, err, nil)
		}
		return errors.New(loggerErrorProjectUrlEmpty)
	}

	resp, sendErr := n.sendRequest(url, req, NotificationActionPayment)

	if sendErr != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, sendErr, nil)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		if n.order.PrivateStatus == recurringpb.OrderStatusPaymentSystemComplete {
			order.PrivateStatus = recurringpb.OrderStatusProjectComplete
		}
	} else {
		zap.S().Errorw(errorNotSuccessStatus, "status", resp.StatusCode, "retry_count", n.RetryCount, "order.uuid", n.order.Uuid)
		if n.RetryCount < RetryMaxCount {
			return n.handleErrorWithRetry(loggerErrorNotificationRetry, errors.New(errorNotSuccessStatus), nil)
		}
		order.PrivateStatus = recurringpb.OrderStatusProjectReject
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

	res.Live = n.order.Project.Status == billingpb.ProjectStatusInProduction

	return res, nil
}

func (n *Default) getSignature(req []byte) string {
	h := sha256.New()
	h.Write([]byte(string(req) + n.order.GetProject().GetSecretKey()))

	return hex.EncodeToString(h.Sum(nil))
}

func (n *Default) getNotificationUrl(_ string) string {
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
