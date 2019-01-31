package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	proto "github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/ProtocolONE/payone-repository/pkg/proto/repository"
	"github.com/ProtocolONE/payone-repository/tools"
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/centrifugal/gocent"
	"github.com/micro/protobuf/ptypes"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"time"
)

const (
	notifierHandlerEmpty   = "default"
	notifierHandlerCardPay = "cardpay"
	notifierHandlerXSolla  = "xsolla"

	errorNotifierHandlerNotFound               = "handler for specified payment system not found"
	errorPaymentMethodUnknown                  = "unknown payment method"
	errorPaymentMethodRequiredTxtParamNotFound = "param \"%s\" not found in DB transaction record\n"
	errorPaymentMethodUnknownStatus            = "unknown transaction status"
	errorEmptyUrl                              = "empty string in url"
	errorNotificationNeedRetry                 = "bad project handler response notification request mark for new send (ID: %s, Action: %s)\n"

	loggerErrorNotificationRetry       = "[PAYONE_NOTIFIER] Project notification failed"
	loggerErrorNotificationUpdate      = "[PAYONE_NOTIFIER] Repository service return error. Update order failed"
	loggerNotificationRetryEnded       = "[PAYONE_NOTIFIER] Republishing message to RabbitMQ ended with max retry count"
	loggerErrorNotificationRetryFailed = "[PAYONE_NOTIFIER] Republish message to RabbitMQ failed"
	LoggerNotificationCentrifugo       = "[PAYONE_NOTIFIER] Send message to centrifugo failed"

	MIMEApplicationJSON = "application/json"

	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"
	HeaderSignature     = "Signature"
	HeaderAuthorization = "Authorization"

	NotificationActionCheck   = "check"
	NotificationActionPayment = "payment"

	centrifugoFieldOrderId = "order_id"
	centrifugoFieldStatus  = "status"
	centrifugoChanelMask   = "payment:notify#%s"

	RetryDlxTimeout   = 600
	RetryExchangeName = "notify-payment-retry"
	RetryMaxCount     = 288
	retryCountHeader  = "x-retry-count"
)

var (
	handlers = map[string]func(*Handler) Notifier{
		notifierHandlerEmpty:   newEmptyHandler,
		notifierHandlerCardPay: newCardPayHandler,
		notifierHandlerXSolla:  newXSollaHandler,
	}
)

type Table map[string]interface{}

type Notifier interface {
	Notify() error
}

type Handler struct {
	order            *proto.Order
	repository       repository.RepositoryService
	logger           *zap.SugaredLogger
	centrifugoClient *gocent.Client

	retBrok    *rabbitmq.Broker
	dlv        amqp.Delivery
	RetryCount int32
}

func NewHandler(
	o *proto.Order,
	rep repository.RepositoryService,
	log *zap.SugaredLogger,
	cClient *gocent.Client,
	retBrok *rabbitmq.Broker,
	dlv amqp.Delivery,
) *Handler {
	rtc := int32(0)

	if v, ok := dlv.Headers[retryCountHeader]; ok {
		rtc = v.(int32)
	}

	return &Handler{
		order:            o,
		repository:       rep,
		logger:           log,
		centrifugoClient: cClient,
		retBrok:          retBrok,
		dlv:              dlv,
		RetryCount:       rtc,
	}
}

func (h *Handler) GetNotifier() (Notifier, error) {
	handler, ok := handlers[h.order.Project.GetCallbackProtocol()]

	if !ok {
		return nil, errors.New(errorNotifierHandlerNotFound)
	}

	m := h.order.GetProject().GetMerchant()

	if m.GetFirstPaymentAt() == nil {
		fpt, err := ptypes.TimestampProto(time.Now())

		if err != nil {
			return nil, err
		}

		m.FirstPaymentAt = fpt
		_, err = h.repository.UpdateMerchant(context.TODO(), m)
	}

	ct, err := ptypes.TimestampProto(time.Now())

	if err != nil {
		return nil, err
	}

	h.order.ProjectLastRequestedAt = ct

	return handler(h), nil
}

func (h *Handler) validateUrl(cUrl string) (*url.URL, error) {
	if cUrl == "" {
		return nil, errors.New(errorEmptyUrl)
	}

	u, err := url.ParseRequestURI(cUrl)

	if err != nil {
		return nil, err
	}

	return u, nil
}

func (h *Handler) request(method, url string, req []byte, headers map[string]string) (*http.Response, error) {
	client := tools.NewLoggedHttpClient(h.logger)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(req))

	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}

	return client.Do(httpReq)
}

func (h *Handler) SendCentrifugoMessage(o *proto.Order) error {
	msg := map[string]interface{}{
		centrifugoFieldOrderId: o.GetId(),
		centrifugoFieldStatus:  OrderAlphabetStatuses[o.GetStatus()],
	}

	ch := fmt.Sprintf(centrifugoChanelMask, msg[centrifugoFieldOrderId])
	b, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	if err = h.centrifugoClient.Publish(context.Background(), ch, b); err != nil {
		return err
	}

	return nil
}

func (h *Handler) HandleError(msg string, err error, t Table) {
	data := []interface{}{
		"error", err.Error(),
		"order_id", h.order.Id,
		"notify_handler", h.order.Project.CallbackProtocol,
	}

	if t != nil && len(t) > 0 {
		for k, v := range t {
			data = append(data, k, v)
		}
	}

	h.logger.Errorw(msg, data...)
}

func (h *Handler) handleErrorWithRetry(msg string, err error, t Table) error {
	h.HandleError(msg, err, t)
	return h.retry()
}

func (h *Handler) retry() (err error) {
	if h.RetryCount >= RetryMaxCount {
		h.logger.Infow(loggerNotificationRetryEnded, "order_id", h.order.Id)
		return
	}

	err = h.retBrok.Publish(h.dlv.RoutingKey, h.order, amqp.Table{"x-retry-count": h.RetryCount + 1})

	if err != nil {
		h.HandleError(loggerErrorNotificationRetryFailed, err, Table{"retry_count": h.RetryCount})
	}

	return
}
