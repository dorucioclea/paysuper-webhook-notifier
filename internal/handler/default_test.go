package handler

import (
	"encoding/json"
	"fmt"
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/globalsign/mgo/bson"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes"
	"github.com/jarcoal/httpmock"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
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

var (
	processUrl    = "http://localhost/process"
	chargebackUrl = "http://localhost/chargeback"
	cancelUrl     = "http://localhost/cancel"
	refundUrl     = "http://localhost/refund"

	projectSecretKey = "some-secret-value"

	dummySignature = "14a691ff433c01ec4181922cb13b161a03f8abe9b65612599fcf34e08213d6cd"
)

type DefaultHandlerTestSuite struct {
	suite.Suite
	redis          *redis.Client
	handler        *Handler
	defaultHandler Notifier
	rtrCnt         int32
}

func Test_DefaultHandler(t *testing.T) {
	suite.Run(t, new(DefaultHandlerTestSuite))
}

func (suite *DefaultHandlerTestSuite) SetupTest() {
	cfg, err := config.NewConfig()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), cfg)
	assert.NotEmpty(suite.T(), cfg.BrokerAddress)

	suite.redis = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
	})

	_, err = suite.redis.Ping().Result()
	assert.NoError(suite.T(), err)

	suite.handler = &Handler{
		order: &billing.Order{
			Id:   bson.NewObjectId().Hex(),
			Uuid: bson.NewObjectId().Hex(),
			Project: &billing.ProjectOrder{
				Id:                   bson.NewObjectId().Hex(),
				Name:                 map[string]string{"en": "test project 1"},
				SecretKey:            projectSecretKey,
				CallbackProtocol:     "default",
				UrlProcessPayment:    processUrl,
				UrlChargebackPayment: chargebackUrl,
				UrlCancelPayment:     cancelUrl,
				UrlRefundPayment:     refundUrl,
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
		},
		repository: mock.NewBillingServerOkMock(),
		redis:      suite.redis,
		cfg:        cfg,
		dlv:        amqp.Delivery{RoutingKey: "*"},
	}

	retryBroker, err := rabbitmq.NewBroker(cfg.BrokerAddress)
	assert.NoError(suite.T(), err)
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

	suite.defaultHandler = newDefaultHandler(suite.handler)
	assert.NotNil(suite.T(), suite.defaultHandler)
}

func (suite *DefaultHandlerTestSuite) TearDownTest() {
	_ = suite.redis.FlushDB()
	_ = suite.redis.Close()
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_Ok() {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", processUrl, httpmock.NewStringResponder(http.StatusOK, ""))

	assert.Equal(suite.T(), suite.handler.order.PrivateStatus, int32(constant.OrderStatusPaymentSystemComplete))

	ps := suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)

	statKey := fmt.Sprintf(psNotificationsKeyMask, suite.handler.order.Id)
	stat, err := suite.handler.getStat(statKey)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), stat.Get(ps))

	nS := suite.handler.order.GetNotificationStatus(ps)
	assert.False(suite.T(), nS)

	err = suite.defaultHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 1)

	assert.Equal(suite.T(), suite.handler.order.PrivateStatus, int32(constant.OrderStatusProjectComplete))

	ps = suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)

	nS = suite.handler.order.GetNotificationStatus(ps)
	assert.True(suite.T(), nS)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_Rejected() {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", processUrl, httpmock.NewStringResponder(http.StatusUnprocessableEntity, ""))

	assert.Equal(suite.T(), suite.handler.order.PrivateStatus, int32(constant.OrderStatusPaymentSystemComplete))

	ps := suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)

	nS := suite.handler.order.GetNotificationStatus(ps)
	assert.False(suite.T(), nS)

	err := suite.defaultHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 1)

	assert.Equal(suite.T(), suite.handler.order.PrivateStatus, int32(constant.OrderStatusProjectReject))

	assert.Equal(suite.T(), suite.handler.order.GetPublicStatus(), constant.OrderPublicStatusRejected)

	nS = suite.handler.order.GetNotificationStatus(ps)
	assert.True(suite.T(), nS)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_DeletedProject_Fail() {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", processUrl, httpmock.NewStringResponder(http.StatusOK, ""))

	suite.handler.order.Project.Status = pkg.ProjectStatusDeleted
	err := suite.defaultHandler.Notify()
	assert.EqualError(suite.T(), err, loggerErrorDeletedProject)
	assert.False(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 0)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_AlreadySent_Ok() {
	ps := suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)

	statKey := fmt.Sprintf(psNotificationsKeyMask, suite.handler.order.Id)
	err := suite.handler.setStat(statKey, ps, true)
	assert.NoError(suite.T(), err)

	stat, err := suite.handler.getStat(statKey)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), stat.Get(ps))

	nS := suite.handler.order.GetNotificationStatus(ps)
	assert.False(suite.T(), nS)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", processUrl, httpmock.NewStringResponder(http.StatusOK, ""))

	err = suite.defaultHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 0)

	nS = suite.handler.order.GetNotificationStatus(ps)
	assert.True(suite.T(), nS)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_EmptyProjectUrl_Fail() {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", processUrl, httpmock.NewStringResponder(http.StatusOK, ""))

	suite.handler.order.Project.UrlProcessPayment = ""
	err := suite.defaultHandler.Notify()
	assert.EqualError(suite.T(), err, loggerErrorProjectUrlEmpty)
	assert.False(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 0)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_getPaymentNotification_Fail() {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", processUrl, httpmock.NewStringResponder(http.StatusOK, ""))

	suite.handler.order.PrivateStatus = 123
	err := suite.defaultHandler.Notify()
	assert.EqualError(suite.T(), err, loggerErrorNotificationMalformed)
	assert.False(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 0)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_Notify_UpdateOrderError() {
	suite.handler.repository = mock.NewBillingServerErrorMock()
	err := suite.defaultHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), suite.handler.retryProcess)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_getNotificationEventName() {
	defaultHandler := &Default{}
	en := defaultHandler.getNotificationEventName("")
	assert.Empty(suite.T(), en)

	ps := suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)

	en = defaultHandler.getNotificationEventName(ps)
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, orderPublicStatusToEventNameMapping[ps])
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_getNotificationUrl() {
	defaultHandler := &Default{}
	defaultHandler.Handler = suite.handler

	en := defaultHandler.getNotificationUrl("")
	assert.Empty(suite.T(), en)

	ps := suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)
	assert.Equal(suite.T(), ps, "processed")

	en = defaultHandler.getNotificationUrl(ps)
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlProcessPayment)
	assert.Equal(suite.T(), en, processUrl)

	en = defaultHandler.getNotificationUrl("chargeback")
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlChargebackPayment)
	assert.Equal(suite.T(), en, chargebackUrl)

	en = defaultHandler.getNotificationUrl("canceled")
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlCancelPayment)
	assert.Equal(suite.T(), en, cancelUrl)

	en = defaultHandler.getNotificationUrl("refunded")
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlRefundPayment)
	assert.Equal(suite.T(), en, refundUrl)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_getSignature() {
	defaultHandler := &Default{}
	defaultHandler.Handler = suite.handler

	req := &OrderNotificationMessage{}
	b, err := json.Marshal(req)
	assert.NoError(suite.T(), err)

	s := defaultHandler.getSignature(b)
	assert.Equal(suite.T(), s, dummySignature)
}

func (suite *DefaultHandlerTestSuite) TestDefaultHandler_sendRequest_signatureCheck_Ok() {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", processUrl, func(req *http.Request) (*http.Response, error) {
		assert.Equal(suite.T(), req.Header.Get(HeaderContentType), MIMEApplicationJSON)
		assert.Equal(suite.T(), req.Header.Get(HeaderAccept), MIMEApplicationJSON)
		assert.Equal(suite.T(), req.Header.Get(HeaderAuthorization), "Signature "+dummySignature)
		return httpmock.NewStringResponse(http.StatusOK, ""), nil
	})

	defaultHandler := &Default{}
	defaultHandler.Handler = suite.handler

	_, err := defaultHandler.sendRequest(processUrl, &OrderNotificationMessage{}, "")
	assert.NoError(suite.T(), err)
}
