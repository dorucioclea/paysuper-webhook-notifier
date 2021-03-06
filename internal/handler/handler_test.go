package handler

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-proto/go/billingpb"
	billMocks "github.com/paysuper/paysuper-proto/go/billingpb/mocks"
	"github.com/paysuper/paysuper-proto/go/recurringpb"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/mock"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	rabbitmq "gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
	"net/http"
	"testing"
)

type HandlerTestSuite struct {
	suite.Suite
	handler    *Handler
	httpClient *http.Client

	logObserver *zap.Logger
	zapRecorder *observer.ObservedLogs
}

func Test_Handler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) SetupTest() {
	cfg, err := config.NewConfig()

	if err != nil {
		assert.FailNow(suite.T(), "Configuration load failed", "%v", err)
	}

	order := &billingpb.Order{
		Id:            "254e3736-000f-5000-8000-178d1d80bf70",
		Uuid:          "254e3736-000f-5000-8000-178d1d80bf70",
		Transaction:   "254e3736-000f-5000-8000-178d1d80bf70",
		Object:        "order",
		Status:        "processed",
		PrivateStatus: 4,
		Description:   "Payment by order",
		CreatedAt:     ptypes.TimestampNow(),
		UpdatedAt:     ptypes.TimestampNow(),
		ReceiptEmail:  "test@unit.test",
		Issuer: &billingpb.OrderIssuer{
			Url:      "http://localhost",
			Embedded: false,
		},
		TotalPaymentAmount: 10.00,
		Currency:           "RUB",
		User: &billingpb.OrderUser{
			Id:     "254e3736-000f-5000-8000-178d1d80bf70",
			Object: "user",
			Email:  "test@unit.test",
			Ip:     "127.0.0.1",
			Address: &billingpb.OrderBillingAddress{
				Country:    "RU",
				City:       "St Petersburg",
				PostalCode: "190000",
				State:      "SPE",
			},
			TechEmail: "eqpAR7uqwC2KBfKZOAEknnKlLcCXtAdn@paysuper.com",
		},
		BillingAddress: &billingpb.OrderBillingAddress{
			Country: "RU",
		},
		Tax: &billingpb.OrderTax{
			Type:     "vat",
			Rate:     0.0,
			Amount:   0.0,
			Currency: "RUB",
		},
		PaymentMethod: &billingpb.PaymentMethodOrder{
			Id:         "254e3736-000f-5000-8000-178d1d80bf70",
			Name:       "Bank card",
			ExternalId: "BANKCARD",
		},
		Project: &billingpb.ProjectOrder{
			Id:                "254e3736-000f-5000-8000-178d1d80bf70",
			MerchantId:        "254e3736-000f-5000-8000-178d1d80bf70",
			Name:              map[string]string{"ru": "Test", "en": "Test"},
			SecretKey:         "Unit Test",
			UrlCheckAccount:   "http://localhost",
			UrlProcessPayment: "http://localhost",
			CallbackProtocol:  "empty",
			Status:            0,
		},
		ProjectOrderId:             "254e3736-000f-5000-8000-178d1d80bf70",
		ProjectAccount:             "test@unit.test",
		PaymentMethodOrderClosedAt: ptypes.TimestampNow(),
		PaymentMethodPayerAccount:  "400000...0002",
		PaymentMethodTxnParams: map[string]string{
			"pan":              "400000...0002",
			"card_holder":      "UNIT TEST",
			"emission_country": "US",
			"token":            "",
			"rrn":              "",
			"is_3ds":           "1",
		},
		PaymentRequisites: map[string]string{
			"bank_issuer_country": "RUSSIA",
			"pan":                 "400000******0002",
			"month":               "12",
			"year":                "2019",
			"card_brand":          "VISA",
			"card_type":           "CREDIT",
			"card_category":       "",
			"bank_issuer_name":    "",
		},
		Type: "order",
	}

	suite.httpClient = mock.NewCentrifugoTransportStatusOk()
	centrifugoPaymentForm := NewCentrifugo(cfg.CentrifugoPaymentForm, suite.httpClient)
	centrifugoDashboard := NewCentrifugo(cfg.CentrifugoDashboard, suite.httpClient)

	redisCl := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
	})

	_, err = redisCl.Ping().Result()

	if err != nil {
		assert.FailNow(suite.T(), "Redis client init failed", "%v", err)
	}

	bs := &billMocks.BillingService{}
	bs.On("UpdateOrder", mock2.Anything, mock2.Anything, mock2.Anything).Return(&billingpb.EmptyResponse{}, nil)

	suite.handler = NewHandler(
		order,
		bs,
		mock.NewBrokerMockOk(),
		mock.NewBrokerMockOk(),
		mock.NewBrokerMockOk(),
		redisCl,
		amqp.Delivery{Headers: amqp.Table{retryCountHeader: int32(1)}},
		cfg,
		centrifugoPaymentForm,
		centrifugoDashboard,
	)

	assert.IsType(suite.T(), &Handler{}, suite.handler)
	assert.IsType(suite.T(), &billingpb.Order{}, suite.handler.order)
	assert.Implements(suite.T(), (*billingpb.BillingService)(nil), suite.handler.repository)
	assert.Implements(suite.T(), (*CentrifugoInterface)(nil), suite.handler.centrifugoPaymentForm)
	assert.Implements(suite.T(), (*CentrifugoInterface)(nil), suite.handler.centrifugoDashboard)
	assert.Implements(suite.T(), (*rabbitmq.BrokerInterface)(nil), suite.handler.retBrok)
	assert.Implements(suite.T(), (*rabbitmq.BrokerInterface)(nil), suite.handler.taxjarTransactionsBroker)
	assert.Implements(suite.T(), (*rabbitmq.BrokerInterface)(nil), suite.handler.taxjarRefundsBroker)
	assert.IsType(suite.T(), amqp.Delivery{}, suite.handler.dlv)
	assert.IsType(suite.T(), &redis.Client{}, suite.handler.redis)
	assert.IsType(suite.T(), &config.Config{}, suite.handler.cfg)
	assert.Equal(suite.T(), int32(1), suite.handler.RetryCount)

	var core zapcore.Core

	lvl := zap.NewAtomicLevel()
	core, suite.zapRecorder = observer.New(lvl)
	suite.logObserver = zap.New(core)
}

func (suite *HandlerTestSuite) TearDownTest() {}

func (suite *HandlerTestSuite) TestHandler_SendToUserCentrifugo_SuccessOrder() {
	zap.ReplaceGlobals(suite.logObserver)

	err := suite.handler.SendToUserCentrifugo(suite.handler.order)
	assert.NoError(suite.T(), err)

	messages := suite.zapRecorder.All()
	assert.NotEmpty(suite.T(), messages)
	assert.Regexp(suite.T(), "payment_form", messages[0].Message)

	msg := make(map[string]interface{})
	err = json.Unmarshal(messages[0].Context[1].Interface.([]byte), &msg)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), msg)

	assert.Contains(suite.T(), msg, "method")
	assert.Equal(suite.T(), msg["method"], "publish")

	assert.Contains(suite.T(), msg, "params")
	assert.NotEmpty(suite.T(), msg["params"])
	assert.Contains(suite.T(), msg["params"], "channel")
	assert.Contains(suite.T(), msg["params"], "data")

	params, ok := msg["params"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), fmt.Sprintf(suite.handler.cfg.CentrifugoUserChannel, suite.handler.order.Uuid), params["channel"])

	data, ok := params["data"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Contains(suite.T(), data, centrifugoFieldDecline)
	assert.Contains(suite.T(), data, centrifugoFieldOrderId)
	assert.Contains(suite.T(), data, centrifugoFieldStatus)

	assert.Nil(suite.T(), data[centrifugoFieldDecline])
	assert.Equal(suite.T(), suite.handler.order.Uuid, data[centrifugoFieldOrderId])
	assert.Equal(suite.T(), OrderAlphabetStatuses[suite.handler.order.PrivateStatus], data[centrifugoFieldStatus])
}

func (suite *HandlerTestSuite) TestHandler_SendToUserCentrifugo_DeclineOrder() {
	zap.ReplaceGlobals(suite.logObserver)

	order := &billingpb.Order{
		Id:            "254e3736-000f-5000-8000-178d1d80bf70",
		Uuid:          "254e3736-000f-5000-8000-178d1d80bf70",
		Transaction:   "254e3736-000f-5000-8000-178d1d80bf70",
		Object:        "order",
		Status:        recurringpb.OrderPublicStatusRejected,
		PrivateStatus: recurringpb.OrderStatusPaymentSystemDeclined,
		Description:   "Payment by order",
		CreatedAt:     ptypes.TimestampNow(),
		UpdatedAt:     ptypes.TimestampNow(),
		ReceiptEmail:  "test@unit.test",
		Issuer: &billingpb.OrderIssuer{
			Url:      "http://localhost",
			Embedded: false,
		},
		TotalPaymentAmount: 10.00,
		Currency:           "RUB",
		User: &billingpb.OrderUser{
			Id:     "254e3736-000f-5000-8000-178d1d80bf70",
			Object: "user",
			Email:  "test@unit.test",
			Ip:     "127.0.0.1",
			Address: &billingpb.OrderBillingAddress{
				Country:    "RU",
				City:       "St Petersburg",
				PostalCode: "190000",
				State:      "SPE",
			},
			TechEmail: "eqpAR7uqwC2KBfKZOAEknnKlLcCXtAdn@paysuper.com",
		},
		BillingAddress: &billingpb.OrderBillingAddress{
			Country: "RU",
		},
		Tax: &billingpb.OrderTax{
			Type:     "vat",
			Rate:     0.0,
			Amount:   0.0,
			Currency: "RUB",
		},
		PaymentMethod: &billingpb.PaymentMethodOrder{
			Id:         "254e3736-000f-5000-8000-178d1d80bf70",
			Name:       "Bank card",
			ExternalId: "BANKCARD",
		},
		Project: &billingpb.ProjectOrder{
			Id:                "254e3736-000f-5000-8000-178d1d80bf70",
			MerchantId:        "254e3736-000f-5000-8000-178d1d80bf70",
			Name:              map[string]string{"ru": "Test", "en": "Test"},
			SecretKey:         "Unit Test",
			UrlCheckAccount:   "http://localhost",
			UrlProcessPayment: "http://localhost",
			CallbackProtocol:  "empty",
			Status:            0,
		},
		ProjectOrderId:             "254e3736-000f-5000-8000-178d1d80bf70",
		ProjectAccount:             "test@unit.test",
		PaymentMethodOrderClosedAt: ptypes.TimestampNow(),
		PaymentMethodPayerAccount:  "400000...0002",
		PaymentMethodTxnParams: map[string]string{
			"pan":                                 "400000...0002",
			"card_holder":                         "UNIT TEST",
			"emission_country":                    "US",
			"token":                               "",
			"rrn":                                 "",
			"is_3ds":                              "1",
			billingpb.TxnParamsFieldDeclineCode:   "11",
			billingpb.TxnParamsFieldDeclineReason: "Some reason",
		},
		PaymentRequisites: map[string]string{
			"bank_issuer_country": "RUSSIA",
			"pan":                 "400000******0002",
			"month":               "12",
			"year":                "2019",
			"card_brand":          "VISA",
			"card_type":           "CREDIT",
			"card_category":       "",
			"bank_issuer_name":    "",
		},
		Type: "order",
	}

	err := suite.handler.SendToUserCentrifugo(order)
	assert.NoError(suite.T(), err)

	messages := suite.zapRecorder.All()
	assert.NotEmpty(suite.T(), messages)
	assert.Regexp(suite.T(), "payment_form", messages[0].Message)

	msg := make(map[string]interface{})
	err = json.Unmarshal(messages[0].Context[1].Interface.([]byte), &msg)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), msg)

	assert.Contains(suite.T(), msg, "method")
	assert.Equal(suite.T(), msg["method"], "publish")

	assert.Contains(suite.T(), msg, "params")
	assert.NotEmpty(suite.T(), msg["params"])
	assert.Contains(suite.T(), msg["params"], "channel")
	assert.Contains(suite.T(), msg["params"], "data")

	params, ok := msg["params"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Equal(suite.T(), fmt.Sprintf(suite.handler.cfg.CentrifugoUserChannel, order.Uuid), params["channel"])

	data, ok := params["data"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Contains(suite.T(), data, centrifugoFieldDecline)
	assert.Contains(suite.T(), data, centrifugoFieldOrderId)
	assert.Contains(suite.T(), data, centrifugoFieldStatus)

	decline, ok := data[centrifugoFieldDecline].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Contains(suite.T(), decline, "code")
	assert.Contains(suite.T(), decline, "reason")

	assert.Equal(suite.T(), order.GetPublicDeclineCode(), decline["code"])
	assert.Equal(suite.T(), order.PaymentMethodTxnParams[billingpb.TxnParamsFieldDeclineReason], decline["reason"])
	assert.Equal(suite.T(), order.Uuid, data[centrifugoFieldOrderId])
	assert.Equal(suite.T(), OrderAlphabetStatuses[order.PrivateStatus], data[centrifugoFieldStatus])
}

func (suite *HandlerTestSuite) TestHandler_sendToAdminCentrifugo_Ok() {
	zap.ReplaceGlobals(suite.logObserver)

	err := suite.handler.sendToAdminCentrifugo(suite.handler.order, "some error")
	assert.NoError(suite.T(), err)

	messages := suite.zapRecorder.All()
	assert.NotEmpty(suite.T(), messages)
	assert.Regexp(suite.T(), "dashboard", messages[0].Message)

	msg := make(map[string]interface{})
	err = json.Unmarshal(messages[0].Context[1].Interface.([]byte), &msg)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), msg)

	assert.Contains(suite.T(), msg, "method")
	assert.Equal(suite.T(), msg["method"], "publish")

	assert.Contains(suite.T(), msg, "params")
	assert.NotEmpty(suite.T(), msg["params"])
	assert.Contains(suite.T(), msg["params"], "channel")
	assert.Contains(suite.T(), msg["params"], "data")

	params, ok := msg["params"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Equal(suite.T(), suite.handler.cfg.CentrifugoAdminChannel, params["channel"])

	data, ok := params["data"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Contains(suite.T(), data, centrifugoFieldCustomMessage)
	assert.Contains(suite.T(), data, centrifugoFieldOrderId)
	assert.Equal(suite.T(), suite.handler.order.Uuid, data[centrifugoFieldOrderId])
	assert.Equal(suite.T(), "some error", data[centrifugoFieldCustomMessage])
}
