package handler

import (
	"fmt"
	"github.com/centrifugal/gocent"
	"github.com/globalsign/mgo/bson"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/mock"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	rabbitmq "gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
	"net/http"
	"testing"
)

type HandlerTestSuite struct {
	suite.Suite
	handler    *Handler
	httpClient *http.Client
}

func Test_Handler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) SetupTest() {
	cfg, err := config.NewConfig()

	if err != nil {
		assert.FailNow(suite.T(), "Configuration load failed", "%v", err)
	}

	order := &billing.Order{
		Id:            bson.NewObjectId().Hex(),
		Uuid:          bson.NewObjectId().Hex(),
		Transaction:   bson.NewObjectId().Hex(),
		Object:        "order",
		Status:        "processed",
		PrivateStatus: 4,
		Description:   "Payment by order",
		CreatedAt:     ptypes.TimestampNow(),
		UpdatedAt:     ptypes.TimestampNow(),
		ReceiptEmail:  "test@unit.test",
		Issuer: &billing.OrderIssuer{
			Url:      "http://localhost",
			Embedded: false,
		},
		TotalPaymentAmount: 10.00,
		Currency:           "RUB",
		User: &billing.OrderUser{
			Id:     bson.NewObjectId().Hex(),
			Object: "user",
			Email:  "test@unit.test",
			Ip:     "127.0.0.1",
			Address: &billing.OrderBillingAddress{
				Country:    "RU",
				City:       "St Petersburg",
				PostalCode: "190000",
				State:      "SPE",
			},
			TechEmail: "eqpAR7uqwC2KBfKZOAEknnKlLcCXtAdn@paysuper.com",
		},
		BillingAddress: &billing.OrderBillingAddress{
			Country: "RU",
		},
		Tax: &billing.OrderTax{
			Type:     "vat",
			Rate:     0.0,
			Amount:   0.0,
			Currency: "RUB",
		},
		PaymentMethod: &billing.PaymentMethodOrder{
			Id:         bson.NewObjectId().Hex(),
			Name:       "Bank card",
			ExternalId: "BANKCARD",
		},
		Project: &billing.ProjectOrder{
			Id:                bson.NewObjectId().Hex(),
			MerchantId:        bson.NewObjectId().Hex(),
			Name:              map[string]string{"ru": "Test", "en": "Test"},
			SecretKey:         "Unit Test",
			UrlCheckAccount:   "http://localhost",
			UrlProcessPayment: "http://localhost",
			CallbackProtocol:  "empty",
			Status:            0,
		},
		ProjectOrderId:             bson.NewObjectId().Hex(),
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

	centCl := gocent.New(
		gocent.Config{
			Addr:       cfg.CentrifugoUrl,
			Key:        cfg.CentrifugoKey,
			HTTPClient: suite.httpClient,
		},
	)

	redisCl := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
	})

	_, err = redisCl.Ping().Result()

	if err != nil {
		assert.FailNow(suite.T(), "Redis client init failed", "%v", err)
	}

	bs := &mock.BillingService{}
	bs.On("UpdateOrder", mock2.Anything, mock2.Anything, mock2.Anything).Return(&grpc.EmptyResponse{}, nil)

	suite.handler = NewHandler(
		order,
		bs,
		centCl,
		mock.NewBrokerMockOk(),
		mock.NewBrokerMockOk(),
		mock.NewBrokerMockOk(),
		redisCl,
		amqp.Delivery{Headers: amqp.Table{retryCountHeader: int32(1)}},
		cfg,
	)

	assert.IsType(suite.T(), &Handler{}, suite.handler)
	assert.IsType(suite.T(), &billing.Order{}, suite.handler.order)
	assert.Implements(suite.T(), (*grpc.BillingService)(nil), suite.handler.repository)
	assert.IsType(suite.T(), &gocent.Client{}, suite.handler.centrifugoClient)
	assert.Implements(suite.T(), (*rabbitmq.BrokerInterface)(nil), suite.handler.retBrok)
	assert.Implements(suite.T(), (*rabbitmq.BrokerInterface)(nil), suite.handler.taxjarTransactionsBroker)
	assert.Implements(suite.T(), (*rabbitmq.BrokerInterface)(nil), suite.handler.taxjarRefundsBroker)
	assert.IsType(suite.T(), amqp.Delivery{}, suite.handler.dlv)
	assert.IsType(suite.T(), &redis.Client{}, suite.handler.redis)
	assert.IsType(suite.T(), &config.Config{}, suite.handler.cfg)
	assert.Equal(suite.T(), int32(1), suite.handler.RetryCount)
}

func (suite *HandlerTestSuite) TearDownTest() {}

func (suite *HandlerTestSuite) TestHandler_SendToUserCentrifugo_SuccessOrder() {
	err := suite.handler.SendToUserCentrifugo(suite.handler.order)
	assert.NoError(suite.T(), err)

	typedHttpClient, ok := suite.httpClient.Transport.(*mock.CentrifugoTransportStatusOk)
	assert.True(suite.T(), ok)

	assert.NotEmpty(suite.T(), typedHttpClient.Msg)
	assert.Contains(suite.T(), typedHttpClient.Msg, "method")
	assert.Equal(suite.T(), typedHttpClient.Msg["method"], "publish")

	assert.Contains(suite.T(), typedHttpClient.Msg, "params")
	assert.NotEmpty(suite.T(), typedHttpClient.Msg["params"])
	assert.Contains(suite.T(), typedHttpClient.Msg["params"], "channel")
	assert.Contains(suite.T(), typedHttpClient.Msg["params"], "data")

	params, ok := typedHttpClient.Msg["params"].(map[string]interface{})
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
	order := &billing.Order{
		Id:            bson.NewObjectId().Hex(),
		Uuid:          bson.NewObjectId().Hex(),
		Transaction:   bson.NewObjectId().Hex(),
		Object:        "order",
		Status:        constant.OrderPublicStatusRejected,
		PrivateStatus: constant.OrderStatusPaymentSystemDeclined,
		Description:   "Payment by order",
		CreatedAt:     ptypes.TimestampNow(),
		UpdatedAt:     ptypes.TimestampNow(),
		ReceiptEmail:  "test@unit.test",
		Issuer: &billing.OrderIssuer{
			Url:      "http://localhost",
			Embedded: false,
		},
		TotalPaymentAmount: 10.00,
		Currency:           "RUB",
		User: &billing.OrderUser{
			Id:     bson.NewObjectId().Hex(),
			Object: "user",
			Email:  "test@unit.test",
			Ip:     "127.0.0.1",
			Address: &billing.OrderBillingAddress{
				Country:    "RU",
				City:       "St Petersburg",
				PostalCode: "190000",
				State:      "SPE",
			},
			TechEmail: "eqpAR7uqwC2KBfKZOAEknnKlLcCXtAdn@paysuper.com",
		},
		BillingAddress: &billing.OrderBillingAddress{
			Country: "RU",
		},
		Tax: &billing.OrderTax{
			Type:     "vat",
			Rate:     0.0,
			Amount:   0.0,
			Currency: "RUB",
		},
		PaymentMethod: &billing.PaymentMethodOrder{
			Id:         bson.NewObjectId().Hex(),
			Name:       "Bank card",
			ExternalId: "BANKCARD",
		},
		Project: &billing.ProjectOrder{
			Id:                bson.NewObjectId().Hex(),
			MerchantId:        bson.NewObjectId().Hex(),
			Name:              map[string]string{"ru": "Test", "en": "Test"},
			SecretKey:         "Unit Test",
			UrlCheckAccount:   "http://localhost",
			UrlProcessPayment: "http://localhost",
			CallbackProtocol:  "empty",
			Status:            0,
		},
		ProjectOrderId:             bson.NewObjectId().Hex(),
		ProjectAccount:             "test@unit.test",
		PaymentMethodOrderClosedAt: ptypes.TimestampNow(),
		PaymentMethodPayerAccount:  "400000...0002",
		PaymentMethodTxnParams: map[string]string{
			"pan":                           "400000...0002",
			"card_holder":                   "UNIT TEST",
			"emission_country":              "US",
			"token":                         "",
			"rrn":                           "",
			"is_3ds":                        "1",
			pkg.TxnParamsFieldDeclineCode:   "11",
			pkg.TxnParamsFieldDeclineReason: "Some reason",
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

	typedHttpClient, ok := suite.httpClient.Transport.(*mock.CentrifugoTransportStatusOk)
	assert.True(suite.T(), ok)

	assert.NotEmpty(suite.T(), typedHttpClient.Msg)
	assert.Contains(suite.T(), typedHttpClient.Msg, "method")
	assert.Equal(suite.T(), typedHttpClient.Msg["method"], "publish")

	assert.Contains(suite.T(), typedHttpClient.Msg, "params")
	assert.NotEmpty(suite.T(), typedHttpClient.Msg["params"])
	assert.Contains(suite.T(), typedHttpClient.Msg["params"], "channel")
	assert.Contains(suite.T(), typedHttpClient.Msg["params"], "data")

	params, ok := typedHttpClient.Msg["params"].(map[string]interface{})
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
	assert.Equal(suite.T(), order.PaymentMethodTxnParams[pkg.TxnParamsFieldDeclineReason], decline["reason"])
	assert.Equal(suite.T(), order.Uuid, data[centrifugoFieldOrderId])
	assert.Equal(suite.T(), OrderAlphabetStatuses[order.PrivateStatus], data[centrifugoFieldStatus])
}

func (suite *HandlerTestSuite) TestHandler_sendToAdminCentrifugo_Ok() {
	err := suite.handler.sendToAdminCentrifugo(suite.handler.order, "some error")
	assert.NoError(suite.T(), err)

	typedHttpClient, ok := suite.httpClient.Transport.(*mock.CentrifugoTransportStatusOk)
	assert.True(suite.T(), ok)

	assert.NotEmpty(suite.T(), typedHttpClient.Msg)
	assert.Contains(suite.T(), typedHttpClient.Msg, "method")
	assert.Equal(suite.T(), typedHttpClient.Msg["method"], "publish")

	assert.Contains(suite.T(), typedHttpClient.Msg, "params")
	assert.NotEmpty(suite.T(), typedHttpClient.Msg["params"])
	assert.Contains(suite.T(), typedHttpClient.Msg["params"], "channel")
	assert.Contains(suite.T(), typedHttpClient.Msg["params"], "data")

	params, ok := typedHttpClient.Msg["params"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Equal(suite.T(), suite.handler.cfg.CentrifugoAdminChannel, params["channel"])

	data, ok := params["data"].(map[string]interface{})
	assert.True(suite.T(), ok)

	assert.Contains(suite.T(), data, centrifugoFieldCustomMessage)
	assert.Contains(suite.T(), data, centrifugoFieldOrderId)
	assert.Equal(suite.T(), suite.handler.order.Uuid, data[centrifugoFieldOrderId])
	assert.Equal(suite.T(), "some error", data[centrifugoFieldCustomMessage])
}
