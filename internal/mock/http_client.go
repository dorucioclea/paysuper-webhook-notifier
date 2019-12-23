package mock

import (
	"bytes"
	"context"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type mockContextKey struct {
	name string
}

type CentrifugoTransportStatusOk struct {
	Transport http.RoundTripper
}

func NewCentrifugoTransportStatusOk() *http.Client {
	return &http.Client{
		Transport: &CentrifugoTransportStatusOk{},
	}
}

func (h *CentrifugoTransportStatusOk) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), &mockContextKey{name: "mockRequestStart"}, time.Now())
	req = req.WithContext(ctx)

	var reqBody []byte

	if req.Body != nil {
		reqBody, _ = ioutil.ReadAll(req.Body)
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))

	zap.L().Info(
		req.URL.Path,
		zap.Any("request_headers", req.Header),
		zap.ByteString("request_body", reqBody),
	)

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}
