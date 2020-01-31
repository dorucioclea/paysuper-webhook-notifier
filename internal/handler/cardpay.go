package handler

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-proto/go/recurringpb"
	"net/http"
	"strconv"
	"time"
)

type CardPay Empty

var OrderAlphabetStatuses = map[int32]string{
	recurringpb.OrderStatusNew:                         "NEW",
	recurringpb.OrderStatusPaymentSystemCreate:         "IN_PROGRESS",
	recurringpb.OrderStatusPaymentSystemRejectOnCreate: "DECLINED",
	recurringpb.OrderStatusPaymentSystemReject:         "DECLINED",
	recurringpb.OrderStatusPaymentSystemComplete:       "COMPLETED",
	recurringpb.OrderStatusProjectInProgress:           "COMPLETED",
	recurringpb.OrderStatusProjectComplete:             "COMPLETED",
	recurringpb.OrderStatusProjectPending:              "COMPLETED",
	recurringpb.OrderStatusProjectReject:               "REFUNDED",
	recurringpb.OrderStatusRefund:                      "REFUNDED",
	recurringpb.OrderStatusChargeback:                  "CHARGEBACK_RESOLVED",
	recurringpb.OrderStatusPaymentSystemDeclined:       "DECLINED",
	recurringpb.OrderStatusPaymentSystemCanceled:       "CANCELLED",
}

func newCardPayHandler(h *Handler) Notifier {
	return &CardPay{Handler: h}
}

func (n *CardPay) Notify() error {
	reqUrl, err := n.validateUrl(n.order.GetProject().GetUrlProcessPayment())

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	callback, err := n.getCallbackRequest()

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	b, err := json.Marshal(callback)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	headers := map[string]string{
		HeaderContentType: MIMEApplicationJSON,
		HeaderAccept:      MIMEApplicationJSON,
		HeaderSignature:   n.getSignature(b),
	}

	resp, err := n.request(http.MethodPost, reqUrl.String(), b, headers)

	if err != nil {
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		n.order.PrivateStatus = recurringpb.OrderStatusProjectComplete
		break
	case http.StatusUnprocessableEntity:
		n.order.PrivateStatus = recurringpb.OrderStatusProjectReject
		break
	default:
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	if _, err = n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}

func (n *CardPay) getCallbackRequest() (*recurringpb.CardPayPaymentCallback, error) {
	req := &recurringpb.CardPayPaymentCallback{
		PaymentMethod: n.order.GetPaymentMethod().GetGroup(),
		CallbackTime:  time.Now().Format(recurringpb.PaymentSystemCardPayDateFormat),
		MerchantOrder: &recurringpb.CardPayMerchantOrder{
			Id:          n.order.GetProjectOrderId(),
			Description: n.order.GetDescription(),
		},
		Customer: &recurringpb.CardPayCustomer{
			Id:     n.order.GetProjectAccount(),
			Ip:     n.order.User.GetIp(),
			Email:  n.order.User.GetEmail(),
			Locale: n.order.User.Locale,
		},
	}

	if err := n.setPaymentData(req); err != nil {
		return nil, err
	}

	switch req.PaymentMethod {
	case recurringpb.PaymentSystemGroupAliasBankCard:
		if err := n.setBankCardTransactionParams(req); err != nil {
			return nil, err
		}
		break
	case recurringpb.PaymentSystemGroupAliasQiwi,
		recurringpb.PaymentSystemGroupAliasWebMoney,
		recurringpb.PaymentSystemGroupAliasNeteller,
		recurringpb.PaymentSystemGroupAliasAlipay:
		req.EwalletAccount = &recurringpb.CardPayEWalletAccount{Id: n.order.GetPaymentMethodPayerAccount()}
		break
	case recurringpb.PaymentSystemGroupAliasBitcoin:
		if err := n.setCryptoCurrencyTransactionParams(req); err != nil {
			return nil, err
		}
		break
	default:
		return nil, errors.New(errorPaymentMethodUnknown)
	}

	return req, nil
}

func (n *CardPay) setPaymentData(_ *recurringpb.CardPayPaymentCallback) error {
	var val string
	var ok bool

	params := n.order.GetPaymentMethodTxnParams()

	pd := &recurringpb.CallbackCardPayPaymentData{
		Id:          n.order.GetId(),
		Amount:      n.order.GetOrderAmount(),
		Currency:    n.order.GetCurrency(),
		Description: n.order.GetDescription(),
	}

	if v, err := ptypes.Timestamp(n.order.GetCreatedAt()); err != nil {
		return err
	} else {
		pd.Created = v.Format(recurringpb.PaymentSystemCardPayDateFormat)
	}

	if val, ok = OrderAlphabetStatuses[n.order.PrivateStatus]; !ok {
		return errors.New(errorPaymentMethodUnknownStatus)
	}

	pd.Status = val

	if val, ok = params["is_3ds"]; ok {
		if v, err := strconv.ParseBool(val); err == nil {
			pd.Is_3D = v
		} else {
			return err
		}
	} else {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "is_3ds"))
	}

	if val, ok = params["rrn"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "rrn"))
	}

	pd.Rrn = val

	if val, ok = params["decline_code"]; ok {
		pd.DeclineCode = val
	}

	if val, ok = params["decline_reason"]; ok {
		pd.DeclineReason = val
	}

	return nil
}

func (n *CardPay) setBankCardTransactionParams(req *recurringpb.CardPayPaymentCallback) error {
	var val string
	var ok bool

	ca := &recurringpb.CallbackCardPayBankCardAccount{MaskedPan: n.order.GetPaymentMethodPayerAccount()}
	params := n.order.GetPaymentMethodTxnParams()

	if val, ok = params["card_holder"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "card_holder"))
	}

	ca.Holder = val

	if val, ok = params["emission_country"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "emission_country"))
	}

	ca.IssuingCountryCode = val

	if val, ok = params["token"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "token"))
	}

	ca.Token = val
	req.CardAccount = ca

	return nil
}

func (n *CardPay) setCryptoCurrencyTransactionParams(_ *recurringpb.CardPayPaymentCallback) error {
	var val string
	var ok bool

	cca := &recurringpb.CallbackCardPayCryptoCurrencyAccount{CryptoAddress: n.order.PaymentMethodPayerAccount}
	params := n.order.GetPaymentMethodTxnParams()

	if val, ok = params["transaction_id"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "transaction_id"))
	}

	cca.CryptoTransactionId = val

	if val, ok = params["amount_crypto"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "amount_crypto"))
	}

	cca.PrcAmount = val

	if val, ok = params["currency_crypto"]; !ok {
		return errors.New(fmt.Sprintf(errorPaymentMethodRequiredTxtParamNotFound, "currency_crypto"))
	}

	cca.PrcCurrency = val

	return nil
}

func (n *CardPay) getSignature(req []byte) string {
	h := sha512.New()
	h.Write([]byte(string(req) + n.order.GetProject().GetSecretKey()))

	return hex.EncodeToString(h.Sum(nil))
}
