package mock

import (
	"context"
	"errors"
	"github.com/globalsign/mgo/bson"
	"github.com/micro/go-micro/client"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
)

const (
	SomeError = "some error"
)

type BillingServerOkMock struct{}
type BillingServerErrorMock struct{}

func NewBillingServerOkMock() grpc.BillingService {
	return &BillingServerOkMock{}
}

func NewBillingServerErrorMock() grpc.BillingService {
	return &BillingServerErrorMock{}
}

func (s *BillingServerOkMock) GetProductsForOrder(
	ctx context.Context,
	in *grpc.GetProductsForOrderRequest,
	opts ...client.CallOption,
) (*grpc.ListProductsResponse, error) {
	return &grpc.ListProductsResponse{}, nil
}

func (s *BillingServerOkMock) OrderCreateProcess(
	ctx context.Context,
	in *billing.OrderCreateRequest,
	opts ...client.CallOption,
) (*billing.Order, error) {
	return &billing.Order{}, nil
}

func (s *BillingServerOkMock) PaymentFormJsonDataProcess(
	ctx context.Context,
	in *grpc.PaymentFormJsonDataRequest,
	opts ...client.CallOption,
) (*grpc.PaymentFormJsonDataResponse, error) {
	return &grpc.PaymentFormJsonDataResponse{}, nil
}

func (s *BillingServerOkMock) PaymentCreateProcess(
	ctx context.Context,
	in *grpc.PaymentCreateRequest,
	opts ...client.CallOption,
) (*grpc.PaymentCreateResponse, error) {
	return &grpc.PaymentCreateResponse{}, nil
}

func (s *BillingServerOkMock) PaymentCallbackProcess(
	ctx context.Context,
	in *grpc.PaymentNotifyRequest,
	opts ...client.CallOption,
) (*grpc.PaymentNotifyResponse, error) {
	return &grpc.PaymentNotifyResponse{}, nil
}

func (s *BillingServerOkMock) RebuildCache(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) UpdateOrder(
	ctx context.Context,
	in *billing.Order,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) UpdateMerchant(
	ctx context.Context,
	in *billing.Merchant,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) GetConvertRate(
	ctx context.Context,
	in *grpc.ConvertRateRequest,
	opts ...client.CallOption,
) (*grpc.ConvertRateResponse, error) {
	return &grpc.ConvertRateResponse{}, nil
}

func (s *BillingServerOkMock) GetMerchantBy(
	ctx context.Context,
	in *grpc.GetMerchantByRequest,
	opts ...client.CallOption,
) (*grpc.MerchantGetMerchantResponse, error) {
	return &grpc.MerchantGetMerchantResponse{}, nil
}

func (s *BillingServerOkMock) ListMerchants(
	ctx context.Context,
	in *grpc.MerchantListingRequest,
	opts ...client.CallOption,
) (*grpc.MerchantListingResponse, error) {
	return &grpc.MerchantListingResponse{}, nil
}

func (s *BillingServerOkMock) ChangeMerchant(
	ctx context.Context,
	in *grpc.OnboardingRequest,
	opts ...client.CallOption,
) (*billing.Merchant, error) {
	m := &billing.Merchant{
		User: &billing.MerchantUser{
			Id:    bson.NewObjectId().Hex(),
			Email: "test@unit.test",
		},
		Name:               in.Name,
		AlternativeName:    in.AlternativeName,
		Website:            in.Website,
		Country:            in.Country,
		State:              in.State,
		Zip:                in.Zip,
		City:               in.City,
		Address:            in.Address,
		AddressAdditional:  in.AddressAdditional,
		RegistrationNumber: in.RegistrationNumber,
		TaxId:              in.TaxId,
		Contacts:           in.Contacts,
		Banking: &billing.MerchantBanking{
			Currency: &billing.Currency{
				CodeInt:  643,
				CodeA3:   in.Banking.Currency,
				Name:     &billing.Name{Ru: "Российский рубль", En: "Russian ruble"},
				IsActive: true,
			},
			Name:          in.Banking.Name,
			Address:       in.Banking.Address,
			AccountNumber: in.Banking.AccountNumber,
			Swift:         in.Banking.Swift,
			Details:       in.Banking.Details,
		},
		Status: pkg.MerchantStatusDraft,
	}

	if in.Id != "" {
		m.Id = in.Id
	} else {
		m.Id = bson.NewObjectId().Hex()
	}

	return m, nil
}

func (s *BillingServerOkMock) ChangeMerchantStatus(
	ctx context.Context,
	in *grpc.MerchantChangeStatusRequest,
	opts ...client.CallOption,
) (*billing.Merchant, error) {
	return &billing.Merchant{Id: in.MerchantId, Status: in.Status}, nil
}

func (s *BillingServerOkMock) CreateNotification(
	ctx context.Context,
	in *grpc.NotificationRequest,
	opts ...client.CallOption,
) (*billing.Notification, error) {
	return &billing.Notification{}, nil
}

func (s *BillingServerOkMock) GetNotification(
	ctx context.Context,
	in *grpc.GetNotificationRequest,
	opts ...client.CallOption,
) (*billing.Notification, error) {
	return &billing.Notification{}, nil
}

func (s *BillingServerOkMock) ListNotifications(
	ctx context.Context,
	in *grpc.ListingNotificationRequest,
	opts ...client.CallOption,
) (*grpc.Notifications, error) {
	return &grpc.Notifications{}, nil
}

func (s *BillingServerOkMock) MarkNotificationAsRead(
	ctx context.Context,
	in *grpc.GetNotificationRequest,
	opts ...client.CallOption,
) (*billing.Notification, error) {
	return &billing.Notification{}, nil
}

func (s *BillingServerOkMock) ListMerchantPaymentMethods(
	ctx context.Context,
	in *grpc.ListMerchantPaymentMethodsRequest,
	opts ...client.CallOption,
) (*grpc.ListingMerchantPaymentMethod, error) {
	return &grpc.ListingMerchantPaymentMethod{}, nil
}

func (s *BillingServerOkMock) GetMerchantPaymentMethod(
	ctx context.Context,
	in *grpc.GetMerchantPaymentMethodRequest,
	opts ...client.CallOption,
) (*grpc.GetMerchantPaymentMethodResponse, error) {
	return &grpc.GetMerchantPaymentMethodResponse{}, nil
}

func (s *BillingServerOkMock) ChangeMerchantPaymentMethod(
	ctx context.Context,
	in *grpc.MerchantPaymentMethodRequest,
	opts ...client.CallOption,
) (*grpc.MerchantPaymentMethodResponse, error) {
	return &grpc.MerchantPaymentMethodResponse{
		Status: pkg.ResponseStatusOk,
		Item: &billing.MerchantPaymentMethod{
			PaymentMethod: &billing.MerchantPaymentMethodIdentification{
				Id:   in.PaymentMethod.Id,
				Name: in.PaymentMethod.Name,
			},
			Commission:  in.Commission,
			Integration: in.Integration,
			IsActive:    in.IsActive,
		},
	}, nil
}

func (s *BillingServerOkMock) CreateRefund(
	ctx context.Context,
	in *grpc.CreateRefundRequest,
	opts ...client.CallOption,
) (*grpc.CreateRefundResponse, error) {
	return &grpc.CreateRefundResponse{
		Status: pkg.ResponseStatusOk,
		Item:   &billing.Refund{},
	}, nil
}

func (s *BillingServerOkMock) ListRefunds(
	ctx context.Context,
	in *grpc.ListRefundsRequest,
	opts ...client.CallOption,
) (*grpc.ListRefundsResponse, error) {
	return &grpc.ListRefundsResponse{}, nil
}

func (s *BillingServerOkMock) GetRefund(
	ctx context.Context,
	in *grpc.GetRefundRequest,
	opts ...client.CallOption,
) (*grpc.CreateRefundResponse, error) {
	return &grpc.CreateRefundResponse{
		Status: pkg.ResponseStatusOk,
		Item:   &billing.Refund{},
	}, nil
}

func (s *BillingServerOkMock) ProcessRefundCallback(
	ctx context.Context,
	in *grpc.CallbackRequest,
	opts ...client.CallOption,
) (*grpc.PaymentNotifyResponse, error) {
	return &grpc.PaymentNotifyResponse{
		Status: pkg.ResponseStatusOk,
	}, nil
}

func (s *BillingServerOkMock) PaymentFormLanguageChanged(
	ctx context.Context,
	in *grpc.PaymentFormUserChangeLangRequest,
	opts ...client.CallOption,
) (*grpc.PaymentFormDataChangeResponse, error) {
	return &grpc.PaymentFormDataChangeResponse{
		Status: pkg.ResponseStatusOk,
		Item: &grpc.PaymentFormDataChangeResponseItem{
			UserAddressDataRequired: true,
			UserIpData: &grpc.UserIpData{
				Country: "RU",
				City:    "St.Petersburg",
				Zip:     "190000",
			},
		},
	}, nil
}

func (s *BillingServerOkMock) PaymentFormPaymentAccountChanged(
	ctx context.Context,
	in *grpc.PaymentFormUserChangePaymentAccountRequest,
	opts ...client.CallOption,
) (*grpc.PaymentFormDataChangeResponse, error) {
	return &grpc.PaymentFormDataChangeResponse{
		Status: pkg.ResponseStatusOk,
		Item: &grpc.PaymentFormDataChangeResponseItem{
			UserAddressDataRequired: true,
			UserIpData: &grpc.UserIpData{
				Country: "RU",
				City:    "St.Petersburg",
				Zip:     "190000",
			},
		},
	}, nil
}

func (s *BillingServerOkMock) ProcessBillingAddress(
	ctx context.Context,
	in *grpc.ProcessBillingAddressRequest,
	opts ...client.CallOption,
) (*grpc.ProcessBillingAddressResponse, error) {
	return &grpc.ProcessBillingAddressResponse{
		Status: pkg.ResponseStatusOk,
		Item: &grpc.ProcessBillingAddressResponseItem{
			HasVat:      true,
			Vat:         10,
			Amount:      10,
			TotalAmount: 20,
		},
	}, nil
}

func (s *BillingServerOkMock) ChangeMerchantData(
	ctx context.Context,
	in *grpc.ChangeMerchantDataRequest,
	opts ...client.CallOption,
) (*grpc.ChangeMerchantDataResponse, error) {
	return &grpc.ChangeMerchantDataResponse{}, nil
}

func (s *BillingServerOkMock) SetMerchantS3Agreement(
	ctx context.Context,
	in *grpc.SetMerchantS3AgreementRequest,
	opts ...client.CallOption,
) (*grpc.ChangeMerchantDataResponse, error) {
	return &grpc.ChangeMerchantDataResponse{}, nil
}

func (s *BillingServerOkMock) ChangeProject(
	ctx context.Context,
	in *billing.Project,
	opts ...client.CallOption,
) (*grpc.ChangeProjectResponse, error) {
	return &grpc.ChangeProjectResponse{}, nil
}

func (s *BillingServerOkMock) GetProject(
	ctx context.Context,
	in *grpc.GetProjectRequest,
	opts ...client.CallOption,
) (*grpc.ChangeProjectResponse, error) {
	return &grpc.ChangeProjectResponse{}, nil
}

func (s *BillingServerOkMock) ListProjects(
	ctx context.Context,
	in *grpc.ListProjectsRequest,
	opts ...client.CallOption,
) (*grpc.ListProjectsResponse, error) {
	return &grpc.ListProjectsResponse{}, nil
}

func (s *BillingServerOkMock) DeleteProject(
	ctx context.Context,
	in *grpc.GetProjectRequest,
	opts ...client.CallOption,
) (*grpc.ChangeProjectResponse, error) {
	return &grpc.ChangeProjectResponse{}, nil
}

func (s *BillingServerOkMock) CreateToken(
	ctx context.Context,
	in *grpc.TokenRequest,
	opts ...client.CallOption,
) (*grpc.TokenResponse, error) {
	return &grpc.TokenResponse{}, nil
}

func (s *BillingServerOkMock) CheckProjectRequestSignature(
	ctx context.Context,
	in *grpc.CheckProjectRequestSignatureRequest,
	opts ...client.CallOption,
) (*grpc.CheckProjectRequestSignatureResponse, error) {
	return &grpc.CheckProjectRequestSignatureResponse{}, nil
}

func (s *BillingServerOkMock) GetOrder(ctx context.Context, in *grpc.GetOrderRequest, opts ...client.CallOption) (*billing.Order, error) {
	panic("implement me")
}

func (s *BillingServerOkMock) FindAllOrders(ctx context.Context, in *grpc.ListOrdersRequest, opts ...client.CallOption) (*billing.OrderPaginate, error) {
	panic("implement me")
}

func (s *BillingServerOkMock) IsOrderCanBePaying(ctx context.Context, in *grpc.IsOrderCanBePayingRequest, opts ...client.CallOption) (*grpc.IsOrderCanBePayingResponse, error) {
	panic("implement me")
}

func (s *BillingServerErrorMock) GetProductsForOrder(
	ctx context.Context,
	in *grpc.GetProductsForOrderRequest,
	opts ...client.CallOption,
) (*grpc.ListProductsResponse, error) {
	return &grpc.ListProductsResponse{}, nil
}

func (s *BillingServerErrorMock) OrderCreateProcess(
	ctx context.Context,
	in *billing.OrderCreateRequest,
	opts ...client.CallOption,
) (*billing.Order, error) {
	return &billing.Order{}, nil
}

func (s *BillingServerErrorMock) PaymentFormJsonDataProcess(
	ctx context.Context,
	in *grpc.PaymentFormJsonDataRequest,
	opts ...client.CallOption,
) (*grpc.PaymentFormJsonDataResponse, error) {
	return &grpc.PaymentFormJsonDataResponse{}, nil
}

func (s *BillingServerErrorMock) PaymentCreateProcess(
	ctx context.Context,
	in *grpc.PaymentCreateRequest,
	opts ...client.CallOption,
) (*grpc.PaymentCreateResponse, error) {
	return &grpc.PaymentCreateResponse{}, nil
}

func (s *BillingServerErrorMock) PaymentCallbackProcess(
	ctx context.Context,
	in *grpc.PaymentNotifyRequest,
	opts ...client.CallOption,
) (*grpc.PaymentNotifyResponse, error) {
	return &grpc.PaymentNotifyResponse{}, nil
}

func (s *BillingServerErrorMock) RebuildCache(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerErrorMock) UpdateOrder(
	ctx context.Context,
	in *billing.Order,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) UpdateMerchant(
	ctx context.Context,
	in *billing.Merchant,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerErrorMock) GetConvertRate(
	ctx context.Context,
	in *grpc.ConvertRateRequest,
	opts ...client.CallOption,
) (*grpc.ConvertRateResponse, error) {
	return &grpc.ConvertRateResponse{}, nil
}

func (s *BillingServerErrorMock) GetMerchantBy(
	ctx context.Context,
	in *grpc.GetMerchantByRequest,
	opts ...client.CallOption,
) (*grpc.MerchantGetMerchantResponse, error) {
	return &grpc.MerchantGetMerchantResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) ListMerchants(
	ctx context.Context,
	in *grpc.MerchantListingRequest,
	opts ...client.CallOption,
) (*grpc.MerchantListingResponse, error) {
	return &grpc.MerchantListingResponse{}, nil
}

func (s *BillingServerErrorMock) ChangeMerchant(
	ctx context.Context,
	in *grpc.OnboardingRequest,
	opts ...client.CallOption,
) (*billing.Merchant, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) ChangeMerchantStatus(
	ctx context.Context,
	in *grpc.MerchantChangeStatusRequest,
	opts ...client.CallOption,
) (*billing.Merchant, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) CreateNotification(
	ctx context.Context,
	in *grpc.NotificationRequest,
	opts ...client.CallOption,
) (*billing.Notification, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetNotification(
	ctx context.Context,
	in *grpc.GetNotificationRequest,
	opts ...client.CallOption,
) (*billing.Notification, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) ListNotifications(
	ctx context.Context,
	in *grpc.ListingNotificationRequest,
	opts ...client.CallOption,
) (*grpc.Notifications, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) MarkNotificationAsRead(
	ctx context.Context,
	in *grpc.GetNotificationRequest,
	opts ...client.CallOption,
) (*billing.Notification, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) ListMerchantPaymentMethods(
	ctx context.Context,
	in *grpc.ListMerchantPaymentMethodsRequest,
	opts ...client.CallOption,
) (*grpc.ListingMerchantPaymentMethod, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetMerchantPaymentMethod(
	ctx context.Context,
	in *grpc.GetMerchantPaymentMethodRequest,
	opts ...client.CallOption,
) (*grpc.GetMerchantPaymentMethodResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) ChangeMerchantPaymentMethod(
	ctx context.Context,
	in *grpc.MerchantPaymentMethodRequest,
	opts ...client.CallOption,
) (*grpc.MerchantPaymentMethodResponse, error) {
	return &grpc.MerchantPaymentMethodResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) CreateRefund(
	ctx context.Context,
	in *grpc.CreateRefundRequest,
	opts ...client.CallOption,
) (*grpc.CreateRefundResponse, error) {
	return &grpc.CreateRefundResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) ListRefunds(
	ctx context.Context,
	in *grpc.ListRefundsRequest,
	opts ...client.CallOption,
) (*grpc.ListRefundsResponse, error) {
	return &grpc.ListRefundsResponse{}, nil
}

func (s *BillingServerErrorMock) GetRefund(
	ctx context.Context,
	in *grpc.GetRefundRequest,
	opts ...client.CallOption,
) (*grpc.CreateRefundResponse, error) {
	return &grpc.CreateRefundResponse{
		Status:  pkg.ResponseStatusNotFound,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) ProcessRefundCallback(
	ctx context.Context,
	in *grpc.CallbackRequest,
	opts ...client.CallOption,
) (*grpc.PaymentNotifyResponse, error) {
	return &grpc.PaymentNotifyResponse{
		Status: pkg.ResponseStatusNotFound,
		Error:  SomeError,
	}, nil
}

func (s *BillingServerErrorMock) PaymentFormLanguageChanged(
	ctx context.Context,
	in *grpc.PaymentFormUserChangeLangRequest,
	opts ...client.CallOption,
) (*grpc.PaymentFormDataChangeResponse, error) {
	return &grpc.PaymentFormDataChangeResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) PaymentFormPaymentAccountChanged(
	ctx context.Context,
	in *grpc.PaymentFormUserChangePaymentAccountRequest,
	opts ...client.CallOption,
) (*grpc.PaymentFormDataChangeResponse, error) {
	return &grpc.PaymentFormDataChangeResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) ProcessBillingAddress(
	ctx context.Context,
	in *grpc.ProcessBillingAddressRequest,
	opts ...client.CallOption,
) (*grpc.ProcessBillingAddressResponse, error) {
	return &grpc.ProcessBillingAddressResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) ChangeMerchantData(
	ctx context.Context,
	in *grpc.ChangeMerchantDataRequest,
	opts ...client.CallOption,
) (*grpc.ChangeMerchantDataResponse, error) {
	return &grpc.ChangeMerchantDataResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) SetMerchantS3Agreement(
	ctx context.Context,
	in *grpc.SetMerchantS3AgreementRequest,
	opts ...client.CallOption,
) (*grpc.ChangeMerchantDataResponse, error) {
	return &grpc.ChangeMerchantDataResponse{
		Status:  pkg.ResponseStatusBadData,
		Message: SomeError,
	}, nil
}

func (s *BillingServerErrorMock) ChangeProject(
	ctx context.Context,
	in *billing.Project,
	opts ...client.CallOption,
) (*grpc.ChangeProjectResponse, error) {
	return &grpc.ChangeProjectResponse{}, nil
}

func (s *BillingServerErrorMock) GetProject(
	ctx context.Context,
	in *grpc.GetProjectRequest,
	opts ...client.CallOption,
) (*grpc.ChangeProjectResponse, error) {
	return &grpc.ChangeProjectResponse{}, nil
}

func (s *BillingServerErrorMock) ListProjects(
	ctx context.Context,
	in *grpc.ListProjectsRequest,
	opts ...client.CallOption,
) (*grpc.ListProjectsResponse, error) {
	return &grpc.ListProjectsResponse{}, nil
}

func (s *BillingServerErrorMock) DeleteProject(
	ctx context.Context,
	in *grpc.GetProjectRequest,
	opts ...client.CallOption,
) (*grpc.ChangeProjectResponse, error) {
	return &grpc.ChangeProjectResponse{}, nil
}

func (s *BillingServerErrorMock) CreateToken(
	ctx context.Context,
	in *grpc.TokenRequest,
	opts ...client.CallOption,
) (*grpc.TokenResponse, error) {
	return &grpc.TokenResponse{}, nil
}

func (s *BillingServerErrorMock) CheckProjectRequestSignature(
	ctx context.Context,
	in *grpc.CheckProjectRequestSignatureRequest,
	opts ...client.CallOption,
) (*grpc.CheckProjectRequestSignatureResponse, error) {
	return &grpc.CheckProjectRequestSignatureResponse{}, nil
}

func (s *BillingServerOkMock) CreateOrUpdateProduct(ctx context.Context, in *grpc.Product, opts ...client.CallOption) (*grpc.Product, error) {
	return &grpc.Product{}, nil
}

func (s *BillingServerOkMock) ListProducts(ctx context.Context, in *grpc.ListProductsRequest, opts ...client.CallOption) (*grpc.ListProductsResponse, error) {
	return &grpc.ListProductsResponse{}, nil
}

func (s *BillingServerOkMock) GetProduct(ctx context.Context, in *grpc.RequestProduct, opts ...client.CallOption) (*grpc.Product, error) {
	return &grpc.Product{}, nil
}

func (s *BillingServerOkMock) DeleteProduct(ctx context.Context, in *grpc.RequestProduct, opts ...client.CallOption) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerErrorMock) CreateOrUpdateProduct(ctx context.Context, in *grpc.Product, opts ...client.CallOption) (*grpc.Product, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) ListProducts(ctx context.Context, in *grpc.ListProductsRequest, opts ...client.CallOption) (*grpc.ListProductsResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetProduct(ctx context.Context, in *grpc.RequestProduct, opts ...client.CallOption) (*grpc.Product, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) DeleteProduct(ctx context.Context, in *grpc.RequestProduct, opts ...client.CallOption) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetOrder(ctx context.Context, in *grpc.GetOrderRequest, opts ...client.CallOption) (*billing.Order, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) FindAllOrders(ctx context.Context, in *grpc.ListOrdersRequest, opts ...client.CallOption) (*billing.OrderPaginate, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) IsOrderCanBePaying(ctx context.Context, in *grpc.IsOrderCanBePayingRequest, opts ...client.CallOption) (*grpc.IsOrderCanBePayingResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) FindByZipCode(
	ctx context.Context,
	in *grpc.FindByZipCodeRequest,
	opts ...client.CallOption,
) (*grpc.FindByZipCodeResponse, error) {
	return &grpc.FindByZipCodeResponse{}, nil
}

func (s *BillingServerErrorMock) FindByZipCode(
	ctx context.Context,
	in *grpc.FindByZipCodeRequest,
	opts ...client.CallOption,
) (*grpc.FindByZipCodeResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) GetCountriesList(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*billing.CountriesList, error) {
	return &billing.CountriesList{}, nil
}

func (s *BillingServerErrorMock) GetCountriesList(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*billing.CountriesList, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) GetCountry(
	ctx context.Context,
	in *billing.GetCountryRequest,
	opts ...client.CallOption,
) (*billing.Country, error) {
	return &billing.Country{}, nil
}

func (s *BillingServerErrorMock) GetCountry(
	ctx context.Context,
	in *billing.GetCountryRequest,
	opts ...client.CallOption,
) (*billing.Country, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) UpdateCountry(
	ctx context.Context,
	in *billing.Country,
	opts ...client.CallOption,
) (*billing.Country, error) {
	return &billing.Country{}, nil
}

func (s *BillingServerErrorMock) UpdateCountry(
	ctx context.Context,
	in *billing.Country,
	opts ...client.CallOption,
) (*billing.Country, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) GetPriceGroup(
	ctx context.Context,
	in *billing.GetPriceGroupRequest,
	opts ...client.CallOption,
) (*billing.PriceGroup, error) {
	return &billing.PriceGroup{}, nil
}

func (s *BillingServerErrorMock) GetPriceGroup(
	ctx context.Context,
	in *billing.GetPriceGroupRequest,
	opts ...client.CallOption,
) (*billing.PriceGroup, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) UpdatePriceGroup(
	ctx context.Context,
	in *billing.PriceGroup,
	opts ...client.CallOption,
) (*billing.PriceGroup, error) {
	return &billing.PriceGroup{}, nil
}

func (s *BillingServerErrorMock) UpdatePriceGroup(
	ctx context.Context,
	in *billing.PriceGroup,
	opts ...client.CallOption,
) (*billing.PriceGroup, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) SetUserNotifySales(
	ctx context.Context,
	in *grpc.SetUserNotifyRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerErrorMock) SetUserNotifySales(
	ctx context.Context,
	in *grpc.SetUserNotifyRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) SetUserNotifyNewRegion(
	ctx context.Context,
	in *grpc.SetUserNotifyRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerErrorMock) SetUserNotifyNewRegion(
	ctx context.Context,
	in *grpc.SetUserNotifyRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerOkMock) CreateOrUpdatePaymentMethod(
	ctx context.Context,
	in *billing.PaymentMethod,
	opts ...client.CallOption,
) (*grpc.ChangePaymentMethodResponse, error) {
	return &grpc.ChangePaymentMethodResponse{}, nil
}

func (s *BillingServerOkMock) CreateOrUpdatePaymentMethodProductionSettings(
	ctx context.Context,
	in *grpc.ChangePaymentMethodParamsRequest,
	opts ...client.CallOption,
) (*grpc.ChangePaymentMethodParamsResponse, error) {
	return &grpc.ChangePaymentMethodParamsResponse{}, nil
}

func (s *BillingServerOkMock) GetPaymentMethodProductionSettings(
	ctx context.Context,
	in *grpc.GetPaymentMethodProductionSettingsRequest,
	opts ...client.CallOption,
) (*billing.PaymentMethodParams, error) {
	return &billing.PaymentMethodParams{}, nil
}

func (s *BillingServerOkMock) DeletePaymentMethodProductionSettings(
	ctx context.Context,
	in *grpc.GetPaymentMethodProductionSettingsRequest,
	opts ...client.CallOption,
) (*grpc.ChangePaymentMethodParamsResponse, error) {
	return &grpc.ChangePaymentMethodParamsResponse{}, nil
}

func (s *BillingServerOkMock) GetAllPaymentChannelCostSystem(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostSystemList, error) {
	return &billing.PaymentChannelCostSystemList{}, nil
}

func (s *BillingServerOkMock) GetPaymentChannelCostSystem(
	ctx context.Context,
	in *billing.PaymentChannelCostSystemRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostSystem, error) {
	return &billing.PaymentChannelCostSystem{}, nil
}

func (s *BillingServerOkMock) SetPaymentChannelCostSystem(
	ctx context.Context,
	in *billing.PaymentChannelCostSystem,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostSystem, error) {
	return &billing.PaymentChannelCostSystem{}, nil
}

func (s *BillingServerOkMock) DeletePaymentChannelCostSystem(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) GetAllPaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentChannelCostMerchantListRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostMerchantList, error) {
	return &billing.PaymentChannelCostMerchantList{}, nil
}

func (s *BillingServerOkMock) GetPaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentChannelCostMerchantRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostMerchant, error) {
	return &billing.PaymentChannelCostMerchant{}, nil
}

func (s *BillingServerOkMock) SetPaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentChannelCostMerchant,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostMerchant, error) {
	return &billing.PaymentChannelCostMerchant{}, nil
}

func (s *BillingServerOkMock) DeletePaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) GetAllMoneyBackCostSystem(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostSystemList, error) {
	return &billing.MoneyBackCostSystemList{}, nil
}

func (s *BillingServerOkMock) GetMoneyBackCostSystem(
	ctx context.Context,
	in *billing.MoneyBackCostSystemRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostSystem, error) {
	return &billing.MoneyBackCostSystem{}, nil
}

func (s *BillingServerOkMock) SetMoneyBackCostSystem(
	ctx context.Context,
	in *billing.MoneyBackCostSystem,
	opts ...client.CallOption,
) (*billing.MoneyBackCostSystem, error) {
	return &billing.MoneyBackCostSystem{}, nil
}

func (s *BillingServerOkMock) DeleteMoneyBackCostSystem(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) GetAllMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.MoneyBackCostMerchantListRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostMerchantList, error) {
	return &billing.MoneyBackCostMerchantList{}, nil
}

func (s *BillingServerOkMock) GetMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.MoneyBackCostMerchantRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostMerchant, error) {
	return &billing.MoneyBackCostMerchant{}, nil
}

func (s *BillingServerOkMock) SetMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.MoneyBackCostMerchant,
	opts ...client.CallOption,
) (*billing.MoneyBackCostMerchant, error) {
	return &billing.MoneyBackCostMerchant{}, nil
}

func (s *BillingServerOkMock) DeleteMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return &grpc.EmptyResponse{}, nil
}

func (s *BillingServerOkMock) CreateAccountingEntry(
	ctx context.Context,
	in *grpc.CreateAccountingEntryRequest,
	opts ...client.CallOption,
) (*grpc.CreateAccountingEntryRequest, error) {
	return &grpc.CreateAccountingEntryRequest{}, nil
}

func (s *BillingServerErrorMock) CreateOrUpdatePaymentMethod(
	ctx context.Context,
	in *billing.PaymentMethod,
	opts ...client.CallOption,
) (*grpc.ChangePaymentMethodResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) CreateOrUpdatePaymentMethodProductionSettings(
	ctx context.Context,
	in *grpc.ChangePaymentMethodParamsRequest,
	opts ...client.CallOption,
) (*grpc.ChangePaymentMethodParamsResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetPaymentMethodProductionSettings(
	ctx context.Context,
	in *grpc.GetPaymentMethodProductionSettingsRequest,
	opts ...client.CallOption,
) (*billing.PaymentMethodParams, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) DeletePaymentMethodProductionSettings(
	ctx context.Context,
	in *grpc.GetPaymentMethodProductionSettingsRequest,
	opts ...client.CallOption,
) (*grpc.ChangePaymentMethodParamsResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetAllPaymentChannelCostSystem(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostSystemList, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetPaymentChannelCostSystem(
	ctx context.Context,
	in *billing.PaymentChannelCostSystemRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostSystem, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) SetPaymentChannelCostSystem(
	ctx context.Context,
	in *billing.PaymentChannelCostSystem,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostSystem, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) DeletePaymentChannelCostSystem(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetAllPaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentChannelCostMerchantListRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostMerchantList, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetPaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentChannelCostMerchantRequest,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostMerchant, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) SetPaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentChannelCostMerchant,
	opts ...client.CallOption,
) (*billing.PaymentChannelCostMerchant, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) DeletePaymentChannelCostMerchant(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetAllMoneyBackCostSystem(
	ctx context.Context,
	in *grpc.EmptyRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostSystemList, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetMoneyBackCostSystem(
	ctx context.Context,
	in *billing.MoneyBackCostSystemRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostSystem, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) SetMoneyBackCostSystem(
	ctx context.Context,
	in *billing.MoneyBackCostSystem,
	opts ...client.CallOption,
) (*billing.MoneyBackCostSystem, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) DeleteMoneyBackCostSystem(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetAllMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.MoneyBackCostMerchantListRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostMerchantList, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) GetMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.MoneyBackCostMerchantRequest,
	opts ...client.CallOption,
) (*billing.MoneyBackCostMerchant, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) SetMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.MoneyBackCostMerchant,
	opts ...client.CallOption,
) (*billing.MoneyBackCostMerchant, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) DeleteMoneyBackCostMerchant(
	ctx context.Context,
	in *billing.PaymentCostDeleteRequest,
	opts ...client.CallOption,
) (*grpc.EmptyResponse, error) {
	return nil, errors.New(SomeError)
}

func (s *BillingServerErrorMock) CreateAccountingEntry(
	ctx context.Context,
	in *grpc.CreateAccountingEntryRequest,
	opts ...client.CallOption,
) (*grpc.CreateAccountingEntryRequest, error) {
	return nil, errors.New(SomeError)
}
