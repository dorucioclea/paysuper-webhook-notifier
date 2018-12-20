package handler

import (
	"bytes"
	"errors"
	proto "github.com/ProtocolONE/payone-repository/pkg/proto/billing"
	"github.com/ProtocolONE/payone-repository/pkg/proto/repository"
	"github.com/ProtocolONE/payone-repository/tools"
	"go.uber.org/zap"
	"net/http"
	"net/url"
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

	MIMEApplicationJSON = "application/json"

	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"
	HeaderSignature     = "Signature"
	HeaderAuthorization = "Authorization"

	NotificationActionCheck = "check"
	NotificationActionPayment = "payment"
)

var (
	handlers = map[string]func(*Handler) Notifier{
		notifierHandlerEmpty:   newEmptyHandler,
		notifierHandlerCardPay: newCardPayHandler,
		notifierHandlerXSolla:  newXSollaHandler,
	}
)

type Notifier interface {
	Notify()
}

type Handler struct {
	order      *proto.Order
	repository repository.RepositoryService
	logger     *zap.SugaredLogger
}

func NewHandler(o *proto.Order, rep repository.RepositoryService, log *zap.SugaredLogger) *Handler {
	return &Handler{order: o, repository: rep, logger: log}
}

func (h *Handler) GetNotifier() (Notifier, error) {
	handler, ok := handlers[h.order.Project.GetCallbackProtocol()]

	if !ok {
		return nil, errors.New(errorNotifierHandlerNotFound)
	}

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
