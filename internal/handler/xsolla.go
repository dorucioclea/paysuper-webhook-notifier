package handler

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/micro/protobuf/ptypes"
	"github.com/paysuper/paysuper-proto/go/recurringpb"
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
	_, err := n.sendRequest(n.order.GetProject().GetUrlCheckAccount(), n.getCheckNotification(), NotificationActionCheck)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	// do payment notify
	req, err := n.getPaymentNotification()

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	resp, err := n.sendRequest(n.order.GetProject().GetUrlProcessPayment(), req, NotificationActionPayment)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		n.order.PrivateStatus = recurringpb.OrderStatusProjectComplete
	} else {
		// in future in that case must be generating refund request to payment system
		n.order.PrivateStatus = recurringpb.OrderStatusProjectReject
	}

	if _, err := n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}

func (n *XSolla) sendRequest(url string, req interface{}, action string) (*http.Response, error) {
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

func (n *XSolla) getCheckNotification() *recurringpb.XSollaCheckNotification {
	return &recurringpb.XSollaCheckNotification{
		NotificationType: xsollaCheckNotificationType,
		User: &recurringpb.XSollaUser{
			Id:      n.order.GetProjectAccount(),
			Ip:      n.order.User.GetIp(),
			Phone:   n.order.User.GetPhone(),
			Email:   n.order.User.GetEmail(),
			Name:    n.order.ProjectAccount,
			Country: n.order.User.Address.GetCountry(),
		},
	}
}

func (n *XSolla) getPaymentNotification() (*recurringpb.XSollaPaymentNotification, error) {
	tDate, err := ptypes.Timestamp(n.order.GetPaymentMethodOrderClosedAt())

	if err != nil {
		return nil, err
	}

	payoutAmount := n.order.GetOrderAmount()

	if n.order.Tax != nil && n.order.Tax.Amount > 0 {
		payoutAmount -= n.order.Tax.Amount
	}

	pn := &recurringpb.XSollaPaymentNotification{
		NotificationType: xsollaPaymentNotificationType,
		Purchase: &recurringpb.XSollaPurchase{
			Checkout: &recurringpb.XSollaCheckout{
				Currency: n.order.GetCurrency(),
				Amount:   n.order.GetOrderAmount(),
			},
			Total: &recurringpb.XSollaTotal{
				Currency: n.order.GetCurrency(),
				Amount:   n.order.GetOrderAmount(),
			},
		},
		User: &recurringpb.XSollaUser{
			Id:      n.order.GetProjectAccount(),
			Ip:      n.order.User.GetIp(),
			Phone:   n.order.User.GetPhone(),
			Email:   n.order.User.GetEmail(),
			Name:    n.order.ProjectAccount,
			Country: n.order.User.Address.GetCountry(),
		},
		Transaction: &recurringpb.XSollaTransaction{
			Id:            n.order.GetId(),
			ExternalId:    n.order.GetProjectOrderId(),
			PaymentDate:   tDate.Format(recurringpb.PaymentSystemCardPayDateFormat),
			PaymentMethod: n.order.GetPaymentMethod().GetGroup(),
			DryRun:        0,
		},
		PaymentDetails: &recurringpb.XSollaPaymentDetails{
			Payment: &recurringpb.XSollaPayment{
				Currency: n.order.GetCurrency(),
				Amount:   n.order.GetOrderAmount(),
			},
			Vat: &recurringpb.XSollaVat{
				Currency: n.order.GetCurrency(),
				Amount:   n.order.GetOrderAmount(),
			},
		},
		CustomParameters: n.order.GetProjectParams(),
	}

	return pn, nil
}

func (n *XSolla) getSignature(req []byte) string {
	h := sha1.New()
	h.Write([]byte(string(req) + n.order.GetProject().GetSecretKey()))

	return hex.EncodeToString(h.Sum(nil))
}
