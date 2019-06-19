package mock

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

type CentrifugoTransportStatusOk struct {
	Transport http.RoundTripper
	Msg       map[string]interface{}
}

func NewCentrifugoTransportStatusOk() *http.Client {
	return &http.Client{
		Transport: &CentrifugoTransportStatusOk{},
	}
}

func (h *CentrifugoTransportStatusOk) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := ioutil.ReadAll(req.Body)
	err := json.Unmarshal(body, &h.Msg)

	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}
