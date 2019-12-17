package handler

import (
	"context"
	"encoding/json"
	"github.com/centrifugal/gocent"
	"github.com/paysuper/paysuper-webhook-notifier/internal/config"
	"go.uber.org/zap"
	"net/http"
)

type CentrifugoInterface interface {
	Publish(context.Context, string, interface{}) error
}

type Centrifugo struct {
	centrifugoClient *gocent.Client
}

func NewCentrifugo(cfg *config.Centrifugo, httpClient *http.Client) CentrifugoInterface {
	centrifugo := &Centrifugo{
		centrifugoClient: gocent.New(
			gocent.Config{
				Addr:       cfg.URL,
				Key:        cfg.ApiSecret,
				HTTPClient: httpClient,
			},
		),
	}

	return centrifugo
}

func (c *Centrifugo) Publish(ctx context.Context, channel string, msg interface{}) error {
	b, err := json.Marshal(msg)

	if err != nil {
		zap.L().Error(
			"Publish message to centrifugo failed",
			zap.Error(err),
			zap.String("channel", channel),
			zap.Any("message", msg),
		)
		return err
	}

	return c.centrifugoClient.Publish(ctx, channel, b)
}
