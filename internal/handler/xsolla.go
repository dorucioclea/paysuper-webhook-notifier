package handler

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/micro/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	proto "github.com/paysuper/paysuper-recurring-repository/pkg/proto/entity"
	"net/http"
)

const (
	xsollaCheckNotificationType   = "user_validation"
	xsollaPaymentNotificationType = "payment"
)

type XSolla Empty

func newXSollaHandler(h *Handler) Notifier {
	return &XSolla{Handler: h}
}

func (n *XSolla) Notify() error {
	// do check notify
	_, err := n.do(n.order.GetProject().GetUrlCheckAccount(), n.getCheckNotification(), NotificationActionCheck)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	// do payment notify
	req, err := n.getPaymentNotification()

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	resp, err := n.do(n.order.GetProject().GetUrlProcessPayment(), req, NotificationActionPayment)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		n.order.Status = constant.OrderStatusProjectComplete
	} else {
		// in future in that case must be generating refund request to payment system
		n.order.Status = constant.OrderStatusProjectReject
	}

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}

func (n *XSolla) do(url string, req interface{}, action string) (*http.Response, error) {
	reqUrl, err := n.validateUrl(url)

	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		HeaderContentType:   MIMEApplicationJSON,
		HeaderAccept:        MIMEApplicationJSON,
		HeaderAuthorization: "Signature " + n.getSignature(b),
	}

	resp, err := n.request(http.MethodPost, reqUrl.String(), b, headers)

	if err != nil {
		return nil, err
	}

	oId := n.order.GetId()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent &&
		resp.StatusCode != http.StatusUnprocessableEntity {
		return nil, errors.New(fmt.Sprintf(errorNotificationNeedRetry, oId, action))
	}

	return resp, nil
}

func (n *XSolla) getCheckNotification() *proto.XSollaCheckNotification {
	return &proto.XSollaCheckNotification{
		NotificationType: xsollaCheckNotificationType,
		User: &proto.XSollaUser{
			Id:      n.order.GetProjectAccount(),
			Ip:      n.order.GetPayerData().GetIp(),
			Phone:   n.order.GetPayerData().GetPhone(),
			Email:   n.order.GetPayerData().GetEmail(),
			Name:    n.order.ProjectAccount,
			Country: n.order.GetPayerData().GetCountryCodeA2(),
		},
	}
}

func (n *XSolla) getPaymentNotification() (*proto.XSollaPaymentNotification, error) {
	tDate, err := ptypes.Timestamp(n.order.GetPaymentMethodOrderClosedAt())

	if err != nil {
		return nil, err
	}

	payoutAmount := n.order.GetAmountOutMerchantAccountingCurrency() -
		n.order.GetPspFeeAmount().GetAmountMerchantCurrency() - n.order.GetPaymentSystemFeeAmount().AmountMerchantCurrency

	if n.order.VatAmount != nil && n.order.VatAmount.AmountMerchantCurrency > 0 {
			payoutAmount -= n.order.VatAmount.AmountMerchantCurrency
	}

	pn := &proto.XSollaPaymentNotification{
		NotificationType: xsollaPaymentNotificationType,
		Purchase: &proto.XSollaPurchase{
			VirtualCurrency: &proto.XSollaVirtualCurrency{
				Name:     n.order.GetFixedPackage().GetName(),
				Sku:      n.order.GetFixedPackage().GetId(),
				Quantity: 1,
				Currency: n.order.GetFixedPackage().GetCurrency().GetCodeA3(),
				Amount:   n.order.GetFixedPackage().GetPrice(),
			},
			Checkout: &proto.XSollaCheckout{
				Currency: n.order.GetProjectOutcomeCurrency().CodeA3,
				Amount:   n.order.GetProjectOutcomeAmount(),
			},
			VirtualItems: &proto.XSollaVirtualItems{
				Items: []*proto.XSollaItem{
					{
						Sku:    n.order.GetFixedPackage().GetId(),
						Amount: 1,
					},
				},
				Currency: n.order.GetProjectOutcomeCurrency().CodeA3,
				Amount:   n.order.GetProjectOutcomeAmount(),
			},
			Total: &proto.XSollaTotal{
				Currency: n.order.GetProjectOutcomeCurrency().CodeA3,
				Amount:   n.order.GetProjectOutcomeAmount(),
			},
		},
		User: &proto.XSollaUser{
			Id:      n.order.GetProjectAccount(),
			Ip:      n.order.GetPayerData().GetIp(),
			Phone:   n.order.GetPayerData().GetPhone(),
			Email:   n.order.GetPayerData().GetEmail(),
			Name:    n.order.ProjectAccount,
			Country: n.order.GetPayerData().GetCountryCodeA2(),
		},
		Transaction: &proto.XSollaTransaction{
			Id:            n.order.GetId(),
			ExternalId:    n.order.GetProjectOrderId(),
			PaymentDate:   tDate.Format(constant.PaymentSystemCardPayDateFormat),
			PaymentMethod: n.order.GetPaymentMethod().GetGroup(),
			DryRun:        0,
		},
		PaymentDetails: &proto.XSollaPaymentDetails{
			Payment: &proto.XSollaPayment{
				Currency: n.order.GetPaymentMethodIncomeCurrency().CodeA3,
				Amount:   n.order.GetPaymentMethodIncomeAmount(),
			},
			Vat: &proto.XSollaVat{
				Currency: n.order.GetPaymentMethodIncomeCurrency().CodeA3,
				Amount:   n.order.VatAmount.AmountPaymentMethodCurrency,
			},
			Payout: &proto.XSollaPayout{
				Currency: n.order.GetProject().GetMerchant().GetPayoutCurrency().CodeA3,
				Amount:   payoutAmount,
			},
			XsollaFee: &proto.XSollaXsollaFee{
				Currency: n.order.GetProject().GetMerchant().GetPayoutCurrency().CodeA3,
				Amount:   n.order.GetPspFeeAmount().GetAmountMerchantCurrency(),
			},
			PaymentMethodFee: &proto.XSollaPaymentMethodFee{
				Currency: n.order.GetProject().GetMerchant().GetPayoutCurrency().CodeA3,
				Amount:   n.order.GetPaymentSystemFeeAmount().GetAmountMerchantCurrency(),
			},
			RepatriationCommission: &proto.XSollaRepatriationCommission{
				Currency: n.order.GetProject().GetMerchant().GetPayoutCurrency().CodeA3,
				Amount:   n.order.GetToPayerFeeAmount().GetAmountMerchantCurrency(),
			},
		},
		CustomParameters: n.order.GetProjectParams(),
	}

	cReq := &grpc.ConvertRateRequest{
		From: n.order.PaymentMethodOutcomeCurrency.CodeInt,
		To:   n.order.GetProject().GetMerchant().GetPayoutCurrency().CodeInt,
	}

	if cRate, err := n.repository.GetConvertRate(context.TODO(), cReq); err != nil {
		return nil, err
	} else {
		pn.PaymentDetails.PayoutCurrencyRate = cRate.Rate
	}

	return pn, nil
}

func (n *XSolla) getSignature(req []byte) string {
	h := sha1.New()
	h.Write([]byte(string(req) + n.order.GetProject().GetSecretKey()))

	return hex.EncodeToString(h.Sum(nil))
}
