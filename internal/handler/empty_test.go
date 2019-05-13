package handler

import (
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/mock"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type EmptyHandlerTestSuite struct {
	suite.Suite
	handler      *Handler
	emptyHandler Notifier
	rtrCnt       int32
}

func Test_EmptyHandler(t *testing.T) {
	suite.Run(t, new(EmptyHandlerTestSuite))
}

func (suite *EmptyHandlerTestSuite) SetupTest() {
	cfg, err := config.NewConfig()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), cfg)
	assert.NotEmpty(suite.T(), cfg.BrokerAddress)

	suite.handler = &Handler{
		order: &billing.Order{
			Id:   bson.NewObjectId().Hex(),
			Uuid: bson.NewObjectId().Hex(),
			Project: &billing.ProjectOrder{
				Id:               bson.NewObjectId().Hex(),
				Name:             bson.NewObjectId().Hex(),
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
			Status:                             constant.OrderStatusPaymentSystemComplete,
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
		},
		repository: mock.NewBillingServerOkMock(),
	}

	retryBroker, err := rabbitmq.NewBroker(cfg.BrokerAddress)
	retryBroker.Opts.QueueOpts.Args = amqp.Table{
		"x-dead-letter-exchange":    constant.PayOneTopicNotifyPaymentName,
		"x-message-ttl":             int32(RetryDlxTimeout * 1000),
		"x-dead-letter-routing-key": "*",
	}
	retryBroker.Opts.ExchangeOpts.Name = RetryExchangeName + "unit_test"

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retryBroker)

	suite.handler.retBrok = retryBroker
	suite.handler.RetryCount = RetryMaxCount - 1

	suite.emptyHandler = newEmptyHandler(suite.handler)
	assert.NotNil(suite.T(), suite.emptyHandler)
}

func (suite *EmptyHandlerTestSuite) TearDownTest() {}

func (suite *EmptyHandlerTestSuite) TestEmptyHandler_Notify_Ok() {
	err := suite.emptyHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), suite.handler.retryProcess)
}

func (suite *EmptyHandlerTestSuite) TestEmptyHandler_Notify_UpdateOrderError() {
	suite.handler.repository = mock.NewBillingServerErrorMock()
	err := suite.emptyHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), suite.handler.retryProcess)
}
