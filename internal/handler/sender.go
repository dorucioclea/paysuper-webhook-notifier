package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"go.uber.org/zap"
	"net/http"
	"net/url"
)

type HttpSender interface {
	Send (url string, req interface{}, action string, secretKey string) (*http.Response, error)
}

type httpSenderImpl struct {
}

func NewHttpSenderImpl() *httpSenderImpl {
	return &httpSenderImpl{}
}

func (n *httpSenderImpl) Send(url string, req interface{}, action string, secretKey string) (*http.Response, error) {
	reqUrl, err := n.validateUrl(url)

	if err != nil {
		zap.L().Error("validate url failed", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	b, err := json.Marshal(req)

	if err != nil {
		zap.L().Error("marshal json error", zap.Error(err))
		return nil, err
	}

	headers := map[string]string{
		HeaderContentType:   MIMEApplicationJSON,
		HeaderAccept:        MIMEApplicationJSON,
		HeaderAuthorization: "Signature " + n.getSignature(b, secretKey),
	}

	resp, err := n.request(http.MethodPost, reqUrl.String(), b, headers)

	if err != nil {
		zap.L().Error("request error", zap.Error(err), zap.String("url", reqUrl.String()), zap.Any("headers", headers), zap.String("body", string(b)))
		return resp, err
	}

	return resp, nil
}

func (n *httpSenderImpl) getSignature(req []byte, secretKey string) string {
	h := sha256.New()
	h.Write([]byte(string(req) + secretKey))

	return hex.EncodeToString(h.Sum(nil))
}

func (h *httpSenderImpl) request(method, url string, req []byte, headers map[string]string) (*http.Response, error) {
	client := tools.NewLoggedHttpClient(zap.S())
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(req))

	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}

	return client.Do(httpReq)
}

func (h *httpSenderImpl) validateUrl(cUrl string) (*url.URL, error) {
	if cUrl == "" {
		return nil, errors.New(errorEmptyUrl)
	}

	u, err := url.ParseRequestURI(cUrl)

	if err != nil {
		return nil, err
	}

	return u, nil
}
