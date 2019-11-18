package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/centrifugal/gocent"
	"github.com/go-redis/redis"
	"github.com/micro/protobuf/ptypes"
	proto "github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	rabbitmq "gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
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
	errorPaymentMethodUnknown                  = "unknown payment method"
	errorPaymentMethodRequiredTxtParamNotFound = "param \"%s\" not found in DB transaction record\n"
	errorPaymentMethodUnknownStatus            = "unknown transaction status"
	errorEmptyUrl                              = "empty string in url"
	errorNotificationNeedRetry                 = "bad project handler response notification request mark for new send (ID: %s, Action: %s)\n"
	errorCentrifugoNotInit                     = "centrifugo client is not configured"
	errorHttpRequestFailed                     = "http request failed"

	loggerErrorNotificationRetry       = "Project notification failed"
	loggerErrorNotificationUpdate      = "Repository service return error. Update order failed"
	loggerNotificationRetryEnded       = "Republishing message to RabbitMQ ended with max retry count"
	loggerErrorNotificationRetryFailed = "Republish message to RabbitMQ failed"
	LoggerNotificationCentrifugo       = "Send message to centrifugo failed"
	LoggerNotificationRedis            = "Set stat in redis failed"
	loggerErrorNotificationMalformed   = "can't get notification object for order"

	MIMEApplicationJSON = "application/json"

	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"
	HeaderSignature     = "Signature"
	HeaderAuthorization = "Authorization"

	NotificationActionCheck   = "check"
	NotificationActionPayment = "payment"
	NotificationActionCheckUser = "check_user"

	centrifugoFieldOrderId       = "order_id"
	centrifugoFieldCustomMessage = "message"
	centrifugoFieldStatus        = "status"
	centrifugoFieldDecline       = "decline"
	centrifugoFieldResponce      = "responce"
	centrifugoFieldTestCase      = "test_case"

	RetryDlxTimeout   = 600
	RetryExchangeName = "notify-payment-retry"
	RetryMaxCount     = 288
	retryCountHeader  = "x-retry-count"

	taxjarNotificationsKeyMask = "tj:notify:%s"

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

type NotificationStat struct {
	StatKey string
	data    map[string]string
}

func (ns *NotificationStat) Get(key string) bool {
	val, ok := ns.data[key]
	if !ok {
		return false
	}
	return val == "1"
}

type Handler struct {
	order                    *proto.Order
	repository               grpc.BillingService
	centrifugoClient         *gocent.Client
	retBrok                  rabbitmq.BrokerInterface
	taxjarTransactionsBroker rabbitmq.BrokerInterface
	taxjarRefundsBroker      rabbitmq.BrokerInterface
	dlv                      amqp.Delivery
	RetryCount               int32
	retryProcess             bool
	redis                    *redis.Client
	cfg                      *config.Config
	sender                   HttpSender
}

func NewHandler(
	o *proto.Order,
	rep grpc.BillingService,
	cClient *gocent.Client,
	retBrok rabbitmq.BrokerInterface,
	taxjarTransactionsBroker rabbitmq.BrokerInterface,
	taxjarRefundsBroker rabbitmq.BrokerInterface,
	redis *redis.Client,
	dlv amqp.Delivery,
	cfg *config.Config,
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
		cfg:                      cfg,
	}
}

func (h *Handler) GetNotifier() (Notifier, error) {
	handler, ok := handlers[h.order.Project.GetCallbackProtocol()]

	if !ok {
		return nil, errors.New(errorNotifierHandlerNotFound)
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

	// Dont't send notification if the Tax rate is empty (see issue #190343)
	if order.Tax.Rate == 0 {
		return
	}

	var (
		topicName        string
		tjStatus         string
		taxjarStatusName string
		taxjarBroker     rabbitmq.BrokerInterface
	)
	if ps == constant.OrderPublicStatusRefunded {
		taxjarBroker = h.taxjarRefundsBroker
		topicName = constant.TaxjarRefundsTopicName
		tjStatus = "refund"
		taxjarStatusName = constant.TaxjarNotificationStatusRefund
	} else {
		taxjarBroker = h.taxjarTransactionsBroker
		topicName = constant.TaxjarTransactionsTopicName
		tjStatus = "payment"
		taxjarStatusName = constant.TaxjarNotificationStatusPayment
	}

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

func (h *Handler) sendToCentrifugo(msg map[string]interface{}, ch string) error {
	b, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	if h.centrifugoClient == nil {
		return errors.New(errorCentrifugoNotInit)
	}

	return h.centrifugoClient.Publish(context.Background(), ch, b)
}

func (h *Handler) SendToUserCentrifugo(order *proto.Order) error {
	msg := map[string]interface{}{
		centrifugoFieldOrderId: order.GetUuid(),
		centrifugoFieldStatus:  OrderAlphabetStatuses[order.PrivateStatus],
		centrifugoFieldDecline: nil,
	}

	if order.IsDeclined() == true {
		code := order.GetPublicDeclineCode()
		reason := order.GetDeclineReason()

		if code != "" && reason != "" {
			msg[centrifugoFieldDecline] = map[string]string{"code": code, "reason": reason}
		}
	}

	return h.sendToCentrifugo(msg, fmt.Sprintf(h.cfg.CentrifugoUserChannel, order.GetUuid()))
}

func (h *Handler) sendToAdminCentrifugo(order *proto.Order, message string) error {
	msg := map[string]interface{}{
		centrifugoFieldCustomMessage: message,
		centrifugoFieldOrderId:       order.GetUuid(),
	}

	return h.sendToCentrifugo(msg, h.cfg.CentrifugoAdminChannel)
}

func (h *Handler) sendToMerchantTestingCentrifugo(order *proto.Order, testCase string, response *http.Response) error {
	msg := map[string]interface{}{
		centrifugoFieldOrderId:  order.GetUuid(),
		centrifugoFieldResponce: response,
		centrifugoFieldTestCase: testCase,
	}

	return h.sendToCentrifugo(msg, h.cfg.CentrifugoMerchantTestingChannel)
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
		if err := h.sendToAdminCentrifugo(h.order, loggerNotificationRetryEnded); err != nil {
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

func (h *Handler) getStat(key string) (*NotificationStat, error) {
	result := &NotificationStat{
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
