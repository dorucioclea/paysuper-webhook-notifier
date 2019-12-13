package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes"
	"github.com/jarcoal/httpmock"
	"github.com/paysuper/paysuper-billing-server/pkg"
	billMocks "github.com/paysuper/paysuper-billing-server/pkg/mocks"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/mock"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"testing"
)

var (
	processUrl    = "http://localhost/process"
	chargebackUrl = "http://localhost/chargeback"
	cancelUrl     = "http://localhost/cancel"
	refundUrl     = "http://localhost/refund"

	dummySignature = "10e4a4e3432c0dbb49f656cc3621c1d55fafb66149d185995e3affc18047d01a"
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

	bs := &billMocks.BillingService{}
	bs.On("UpdateOrder", mock2.Anything, mock2.Anything, mock2.Anything).Return(&grpc.EmptyResponse{}, nil)

	suite.handler = &Handler{
		order: &billing.Order{
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
				Id:                   bson.NewObjectId().Hex(),
				MerchantId:           bson.NewObjectId().Hex(),
				Name:                 map[string]string{"ru": "Test", "en": "Test"},
				SecretKey:            "Unit Test",
				UrlCheckAccount:      "http://localhost",
				UrlProcessPayment:    processUrl,
				UrlChargebackPayment: chargebackUrl,
				UrlCancelPayment:     cancelUrl,
				UrlRefundPayment:     refundUrl,
				CallbackProtocol:     "default",
				Status:               0,
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
		},
		repository: bs,
		redis:      suite.redis,
		cfg:        cfg,
		dlv:        amqp.Delivery{RoutingKey: "*"},
	}

	suite.handler.centrifugoPaymentForm, suite.handler.centrifugoDashboard = NewCentrifugo(cfg, mock.NewCentrifugoTransportStatusOk())

	suite.handler.retBrok = mock.NewBrokerMockOk()
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
	assert.True(suite.T(), suite.handler.retryProcess)

	info := httpmock.GetCallCountInfo()
	assert.Equal(suite.T(), len(info), 1)
	assert.Equal(suite.T(), info["POST "+processUrl], 1)
	assert.Equal(suite.T(), suite.handler.order.PrivateStatus, int32(constant.OrderStatusPaymentSystemComplete))

	suite.handler.RetryCount = RetryMaxCount
	err = suite.defaultHandler.Notify()
	assert.NoError(suite.T(), err)

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
	bs := &billMocks.BillingService{}
	bs.On("UpdateOrder", mock2.Anything, mock2.Anything, mock2.Anything).Return(nil, errors.New("some error"))

	suite.handler.repository = bs
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

	ps := suite.handler.order.GetPublicStatus()
	assert.Equal(suite.T(), ps, constant.OrderPublicStatusProcessed)
	assert.Equal(suite.T(), ps, "processed")

	en := defaultHandler.getNotificationUrl(ps)
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlProcessPayment)
	assert.Equal(suite.T(), en, processUrl)

	en = defaultHandler.getNotificationUrl("chargeback")
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlProcessPayment)
	assert.Equal(suite.T(), en, processUrl)

	en = defaultHandler.getNotificationUrl("canceled")
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlProcessPayment)
	assert.Equal(suite.T(), en, processUrl)

	en = defaultHandler.getNotificationUrl("refunded")
	assert.NotEmpty(suite.T(), en)
	assert.Equal(suite.T(), en, suite.handler.order.Project.UrlProcessPayment)
	assert.Equal(suite.T(), en, processUrl)
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
