package handler

import (
	"errors"
	"github.com/golang/protobuf/ptypes"
	billMocks "github.com/paysuper/paysuper-billing-server/pkg/mocks"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"github.com/paysuper/paysuper-webhook-notifier/internal/mock"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
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

	bs := &billMocks.BillingService{}
	bs.On("UpdateOrder", mock2.Anything, mock2.Anything, mock2.Anything).Return(&grpc.EmptyResponse{}, nil)

	suite.handler = &Handler{
		order: &billing.Order{
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
			Issuer: &billing.OrderIssuer{
				Url:      "http://localhost",
				Embedded: false,
			},
			TotalPaymentAmount: 10.00,
			Currency:           "RUB",
			User: &billing.OrderUser{
				Id:     "254e3736-000f-5000-8000-178d1d80bf70",
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
				Id:         "254e3736-000f-5000-8000-178d1d80bf70",
				Name:       "Bank card",
				ExternalId: "BANKCARD",
			},
			Project: &billing.ProjectOrder{
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
		},
		repository: bs,
	}

	suite.handler.retBrok = mock.NewBrokerMockOk()
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
	bs := &billMocks.BillingService{}
	bs.On("UpdateOrder", mock2.Anything, mock2.Anything, mock2.Anything).Return(nil, errors.New("some error"))

	suite.handler.repository = bs
	err := suite.emptyHandler.Notify()
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), suite.handler.retryProcess)
}
