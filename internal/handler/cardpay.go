package handler

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	proto "github.com/paysuper/paysuper-recurring-repository/pkg/proto/entity"
	"net/http"
	"strconv"
	"time"
)

type CardPay Empty

var OrderAlphabetStatuses = map[int32]string{
	constant.OrderStatusNew:                         "NEW",
	constant.OrderStatusPaymentSystemCreate:         "IN_PROGRESS",
	constant.OrderStatusPaymentSystemRejectOnCreate: "DECLINED",
	constant.OrderStatusPaymentSystemReject:         "DECLINED",
	constant.OrderStatusPaymentSystemComplete:       "COMPLETED",
	constant.OrderStatusProjectInProgress:           "COMPLETED",
	constant.OrderStatusProjectComplete:             "COMPLETED",
	constant.OrderStatusProjectPending:              "COMPLETED",
	constant.OrderStatusProjectReject:               "REFUNDED",
	constant.OrderStatusRefund:                      "REFUNDED",
	constant.OrderStatusChargeback:                  "CHARGEBACK_RESOLVED",
	constant.OrderStatusPaymentSystemDeclined:       "DECLINED",
	constant.OrderStatusPaymentSystemCanceled:       "CANCELLED",
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
		n.order.PrivateStatus = constant.OrderStatusProjectComplete
		break
	case http.StatusUnprocessableEntity:
		n.order.PrivateStatus = constant.OrderStatusProjectReject
		break
	default:
		return n.handleErrorWithRetry(loggerErrorNotificationRetry, err, nil)
	}

	if _, err = n.repository.UpdateOrder(context.TODO(), n.order); err != nil {
		n.HandleError(loggerErrorNotificationUpdate, err, nil)
	}

	return nil
}

func (n *CardPay) getCallbackRequest() (*proto.CardPayPaymentCallback, error) {
	req := &proto.CardPayPaymentCallback{
		PaymentMethod: n.order.GetPaymentMethod().GetGroup(),
		CallbackTime:  time.Now().Format(constant.PaymentSystemCardPayDateFormat),
		MerchantOrder: &proto.CardPayMerchantOrder{
			Id:          n.order.GetProjectOrderId(),
			Description: n.order.GetDescription(),
		},
		Customer: &proto.CardPayCustomer{
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
	case constant.PaymentSystemGroupAliasBankCard:
		if err := n.setBankCardTransactionParams(req); err != nil {
			return nil, err
		}
		break
	case constant.PaymentSystemGroupAliasQiwi,
		constant.PaymentSystemGroupAliasWebMoney,
		constant.PaymentSystemGroupAliasNeteller,
		constant.PaymentSystemGroupAliasAlipay:
		req.EwalletAccount = &proto.CardPayEWalletAccount{Id: n.order.GetPaymentMethodPayerAccount()}
		break
	case constant.PaymentSystemGroupAliasBitcoin:
		if err := n.setCryptoCurrencyTransactionParams(req); err != nil {
			return nil, err
		}
		break
	default:
		return nil, errors.New(errorPaymentMethodUnknown)
	}

	return req, nil
}

func (n *CardPay) setPaymentData(req *proto.CardPayPaymentCallback) error {
	var val string
	var ok bool

	params := n.order.GetPaymentMethodTxnParams()

	pd := &proto.CallbackCardPayPaymentData{
		Id:          n.order.GetId(),
		Amount:      n.order.GetProjectOutcomeAmount(),
		Currency:    n.order.GetProjectOutcomeCurrency().CodeA3,
		Description: n.order.GetDescription(),
	}

	if v, err := ptypes.Timestamp(n.order.GetCreatedAt()); err != nil {
		return err
	} else {
		pd.Created = v.Format(constant.PaymentSystemCardPayDateFormat)
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

func (n *CardPay) setBankCardTransactionParams(req *proto.CardPayPaymentCallback) error {
	var val string
	var ok bool

	ca := &proto.CallbackCardPayBankCardAccount{MaskedPan: n.order.GetPaymentMethodPayerAccount()}
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

func (n *CardPay) setCryptoCurrencyTransactionParams(req *proto.CardPayPaymentCallback) error {
	var val string
	var ok bool

	cca := &proto.CallbackCardPayCryptoCurrencyAccount{CryptoAddress: n.order.PaymentMethodPayerAccount}
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
