package handler

import (
	"fmt"
	rabbitmq "github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/centrifugal/gocent"
	"github.com/globalsign/mgo/bson"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/mock"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
		Id:   bson.NewObjectId().Hex(),
		Uuid: bson.NewObjectId().Hex(),
		Project: &billing.ProjectOrder{
			Id:               bson.NewObjectId().Hex(),
			Name:             map[string]string{"en": "test project 1"},
			SecretKey:        bson.NewObjectId().Hex(),
			CallbackProtocol: "empty",
		},
		Description:         "some description",
		ProjectOrderId:      bson.NewObjectId().Hex(),
		ProjectAccount:      bson.NewObjectId().Hex(),
		ProjectIncomeAmount: 10,
		ProjectIncomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
		ProjectOutcomeAmount: 10,
		ProjectOutcomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
		PrivateStatus:                      constant.OrderStatusPaymentSystemComplete,
		CreatedAt:                          ptypes.TimestampNow(),
		IsJsonRequest:                      false,
		AmountInMerchantAccountingCurrency: tools.FormatAmount(10),
		PaymentMethodOutcomeAmount:         10,
		PaymentMethodOutcomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
		PaymentMethodIncomeAmount: 10,
		PaymentMethodIncomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
	}

	suite.httpClient = mock.NewCentrifugoTransportStatusOk()

	centCl := gocent.New(
		gocent.Config{
			Addr:       cfg.CentrifugoUrl,
			Key:        cfg.CentrifugoKey,
			HTTPClient: suite.httpClient,
		},
	)

	broker, err := rabbitmq.NewBroker(cfg.BrokerAddress)

	if err != nil {
		assert.FailNow(suite.T(), "RabbitMQ init failed", "%v", err)
	}

	redisCl := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
	})

	_, err = redisCl.Ping().Result()

	if err != nil {
		assert.FailNow(suite.T(), "Redis client init failed", "%v", err)
	}

	suite.handler = NewHandler(
		order,
		mock.NewBillingServerOkMock(),
		centCl,
		broker,
		broker,
		broker,
		redisCl,
		amqp.Delivery{Headers: amqp.Table{retryCountHeader: int32(1)}},
		cfg,
	)

	assert.IsType(suite.T(), &Handler{}, suite.handler)
	assert.IsType(suite.T(), &billing.Order{}, suite.handler.order)
	assert.Implements(suite.T(), (*grpc.BillingService)(nil), suite.handler.repository)
	assert.IsType(suite.T(), &gocent.Client{}, suite.handler.centrifugoClient)
	assert.IsType(suite.T(), &rabbitmq.Broker{}, suite.handler.retBrok)
	assert.IsType(suite.T(), &rabbitmq.Broker{}, suite.handler.taxjarTransactionsBroker)
	assert.IsType(suite.T(), &rabbitmq.Broker{}, suite.handler.taxjarRefundsBroker)
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
		Id:   bson.NewObjectId().Hex(),
		Uuid: bson.NewObjectId().Hex(),
		Project: &billing.ProjectOrder{
			Id:               bson.NewObjectId().Hex(),
			Name:             map[string]string{"en": "test project 1"},
			SecretKey:        bson.NewObjectId().Hex(),
			CallbackProtocol: "empty",
		},
		Description:         "some description",
		ProjectOrderId:      bson.NewObjectId().Hex(),
		ProjectAccount:      bson.NewObjectId().Hex(),
		ProjectIncomeAmount: 10,
		ProjectIncomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
		ProjectOutcomeAmount: 10,
		ProjectOutcomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
		PrivateStatus: constant.OrderStatusPaymentSystemDeclined,
		PaymentMethodTxnParams: map[string]string{
			pkg.TxnParamsFieldDeclineCode:   "11",
			pkg.TxnParamsFieldDeclineReason: "Some reason",
		},
		CreatedAt:                          ptypes.TimestampNow(),
		IsJsonRequest:                      false,
		AmountInMerchantAccountingCurrency: tools.FormatAmount(10),
		PaymentMethodOutcomeAmount:         10,
		PaymentMethodOutcomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
		PaymentMethodIncomeAmount: 10,
		PaymentMethodIncomeCurrency: &billing.Currency{
			CodeInt:  643,
			CodeA3:   "RUB",
			Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
			IsActive: true,
		},
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
	err := suite.handler.sendToAdminCentrifugo(suite.handler.order, mock.SomeError)
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
	assert.Equal(suite.T(), mock.SomeError, data[centrifugoFieldCustomMessage])
}
