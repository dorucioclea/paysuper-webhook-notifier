package mock

import "net/http"

type httpSenderImpl struct {
}

func NewHttpSenderImpl() *httpSenderImpl {
	return &httpSenderImpl{}
}

func (n *httpSenderImpl) Send(url string, req interface{}, action string, secretKey string) (*http.Response, error) {
	return &http.Response{

	}, nil
}

