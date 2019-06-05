package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/centrifugal/gocent"
	"github.com/go-redis/redis"
	"github.com/micro/protobuf/ptypes"
	proto "github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"time"
)

const (
	// Notification request not send, order just mark as successfully complete
	notifierHandlerEmpty = "empty"
	// Notification request send by PaySuper notification protocol
	notifierHandlerDefault = "default"
	// Notification request send by CardPay notification protocol
	notifierHandlerCardPay = "cardpay"
	// Notification request send by XSolla  notification protocol
	notifierHandlerXSolla = "xsolla"

	errorNotifierHandlerNotFound               = "handler for specified payment system not found"
	errorGetMerchantFailed                     = "gRpc call to get merchant failed"
	errorMerchantNotFound                      = "merchant not found"
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
	LoggerNotificationRedis            = "[PAYONE_NOTIFIER] Set stat in redis failed"
	loggerErrorNotificationMalfored    = "[PAYONE_NOTIFIER] Can't get notification object for order"

	MIMEApplicationJSON = "application/json"

	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"
	HeaderSignature     = "Signature"
	HeaderAuthorization = "Authorization"

	NotificationActionCheck   = "check"
	NotificationActionPayment = "payment"

	centrifugoFieldOrderId       = "order_id"
	centrifugoFieldCustomMessage = "message"
	centrifugoFieldStatus        = "status"
	centrifugoChanelMask         = "paysuper:order#%s"

	RetryDlxTimeout   = 600
	RetryExchangeName = "notify-payment-retry"
	RetryMaxCount     = 288
	retryCountHeader  = "x-retry-count"

	taxjarNotificationsKeyMask = "tj:notify:%s"
	taxjarStatusNameMask       = "taxjar.%s"

	CountryCodeUSA = "US"
)

var (
	handlers = map[string]func(*Handler) Notifier{
		notifierHandlerEmpty:   newEmptyHandler,
		notifierHandlerDefault: newDefaultHandler,
		notifierHandlerCardPay: newCardPayHandler,
		notifierHandlerXSolla:  newXSollaHandler,
	}
)

type Table map[string]interface{}

type Notifier interface {
	Notify() error
}

type NoitificationStat struct {
	StatKey string
	data    map[string]string
}

func (ns *NoitificationStat) Get(key string) bool {
	val, ok := ns.data[key]
	if !ok {
		return false
	}
	return val == "true"
}

type Handler struct {
	order            *proto.Order
	repository       grpc.BillingService
	centrifugoClient *gocent.Client

	retBrok                  *rabbitmq.Broker
	taxjarTransactionsBroker *rabbitmq.Broker
	taxjarRefundsBroker      *rabbitmq.Broker
	dlv                      amqp.Delivery
	RetryCount               int32
	retryProcess             bool
	redis                    *redis.Client
}

func NewHandler(
	o *proto.Order,
	rep grpc.BillingService,
	cClient *gocent.Client,
	retBrok *rabbitmq.Broker,
	taxjarTransactionsBroker *rabbitmq.Broker,
	taxjarRefundsBroker *rabbitmq.Broker,
	redis *redis.Client,
	dlv amqp.Delivery,
) *Handler {
	rtc := int32(0)

	if v, ok := dlv.Headers[retryCountHeader]; ok {
		rtc = v.(int32)
	}

	return &Handler{
		order:                    o,
		repository:               rep,
		centrifugoClient:         cClient,
		retBrok:                  retBrok,
		taxjarTransactionsBroker: taxjarTransactionsBroker,
		taxjarRefundsBroker:      taxjarRefundsBroker,
		redis:                    redis,
		dlv:                      dlv,
		RetryCount:               rtc,
	}
}

func (h *Handler) GetMerchant(id string) (*proto.Merchant, error) {
	req := &grpc.GetMerchantByRequest{
		MerchantId: id,
	}
	res, err := h.repository.GetMerchantBy(context.TODO(), req)
	if err != nil {
		return nil, errors.New(errorGetMerchantFailed)
	}

	if res.Item == nil {
		return nil, errors.New(errorMerchantNotFound)
	}

	return res.Item, nil
}

func (h *Handler) GetNotifier() (Notifier, error) {
	handler, ok := handlers[h.order.Project.GetCallbackProtocol()]

	if !ok {
		return nil, errors.New(errorNotifierHandlerNotFound)
	}

	m, err := h.GetMerchant(h.order.Project.MerchantId)
	if err != nil {
		return nil, err
	}

	if m.GetFirstPaymentAt() == nil {
		m.FirstPaymentAt = ptypes.TimestampNow()
		_, err := h.repository.UpdateMerchant(context.TODO(), m)

		if err != nil {
			h.HandleError("gRpc call to update merchant failed", err, Table{"merchant": m})
		}
	}

	h.order.ProjectLastRequestedAt = ptypes.TimestampNow()

	h.trySendToTaxJar()

	return handler(h), nil
}

func (h *Handler) trySendToTaxJar() {
	order := h.order

	ps := order.GetPublicStatus()
	isShouldSend := (ps == constant.OrderPublicStatusProcessed || ps == constant.OrderPublicStatusRefunded) && order.GetCountry() == CountryCodeUSA

	if !isShouldSend {
		return
	}

	var (
		topicName    string
		tjStatus     string
		taxjarBroker *rabbitmq.Broker
	)
	if ps == constant.OrderPublicStatusRefunded {
		taxjarBroker = h.taxjarRefundsBroker
		topicName = constant.TaxjarRefundsTopicName
		tjStatus = "refund"
	} else {
		taxjarBroker = h.taxjarTransactionsBroker
		topicName = constant.TaxjarTransactionsTopicName
		tjStatus = "payment"
	}
	taxjarStatusName := fmt.Sprintf(taxjarStatusNameMask, tjStatus)

	statKey := fmt.Sprintf(taxjarNotificationsKeyMask, order.Id)
	stat, err := h.getStat(statKey)
	if err != nil {
		_ = h.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
		return
	}

	if stat.Get(tjStatus) == true {
		order.SetNotificationStatus(taxjarStatusName, true)
		if _, err := h.repository.UpdateOrder(context.TODO(), order); err != nil {
			h.HandleError(loggerErrorNotificationUpdate, err, nil)
		}
		return
	}

	publishErr := taxjarBroker.Publish(topicName, order, amqp.Table{"x-retry-count": int32(0)})
	isSuccess := publishErr == nil

	err = h.setStat(stat.StatKey, tjStatus, isSuccess)
	if err != nil {
		h.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	order.SetNotificationStatus(taxjarStatusName, true)
	if _, err := h.repository.UpdateOrder(context.TODO(), order); err != nil {
		h.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	if publishErr != nil {
		_ = h.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
		return
	}
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
	client := tools.NewLoggedHttpClient(zap.S())
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(req))

	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}

	return client.Do(httpReq)
}

func (h *Handler) SendCentrifugoMessage(o *proto.Order, message string) error {
	msg := map[string]interface{}{
		centrifugoFieldCustomMessage: message,
		centrifugoFieldOrderId:       o.GetId(),
		centrifugoFieldStatus:        OrderAlphabetStatuses[o.PrivateStatus],
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
		"error", err,
		"order_id", h.order.Id,
		"notify_handler", h.order.Project.CallbackProtocol,
	}

	if t != nil && len(t) > 0 {
		for k, v := range t {
			data = append(data, k, v)
		}
	}

	zap.S().Errorw(msg, data...)
}

func (h *Handler) handleErrorWithRetry(msg string, err error, t Table) error {
	h.HandleError(msg, err, t)
	return h.retry()
}

func (h *Handler) retry() (err error) {
	if h.RetryCount >= RetryMaxCount {
		zap.S().Infow(loggerNotificationRetryEnded, "order_id", h.order.Id)
		if err := h.SendCentrifugoMessage(h.order, loggerNotificationRetryEnded); err != nil {
			h.HandleError(LoggerNotificationCentrifugo, err, nil)
		}
		return
	}

	err = h.retBrok.Publish(h.dlv.RoutingKey, h.order, amqp.Table{"x-retry-count": h.RetryCount + 1})

	if err != nil {
		h.HandleError(loggerErrorNotificationRetryFailed, err, Table{"retry_count": h.RetryCount})
		time.Sleep(5 * time.Second)
		return h.retry()
	}

	h.retryProcess = true
	return
}

func (h *Handler) getStat(key string) (*NoitificationStat, error) {
	result := &NoitificationStat{
		StatKey: key,
	}
	var err error
	result.data, err = h.redis.HGetAll(key).Result()
	if err != nil {
		h.HandleError("get notification stat failed", err, nil)
		return nil, err
	}
	return result, nil
}

func (h *Handler) setStat(key string, field string, val bool) error {
	err := h.redis.HSet(key, field, val).Err()
	if err != nil {
		h.HandleError("set notification stat failed", err, nil)
		return err
	}
	return nil
}
