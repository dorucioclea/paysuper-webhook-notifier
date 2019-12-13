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
	Publish(ctx context.Context, channel string, msg interface{}) error
}

type Centrifugo struct {
	client *gocent.Client
}

func NewCentrifugo(
	cfg *config.Config,
	httpClient *http.Client,
) (centrifugoPaymentForm CentrifugoInterface, centrifugoDashboard CentrifugoInterface) {
	centrifugoPaymentForm = &Centrifugo{
		client: gocent.New(
			gocent.Config{
				Addr:       cfg.CentrifugoURLPaymentForm,
				Key:        cfg.CentrifugoApiSecretPaymentForm,
				HTTPClient: httpClient,
			},
		),
	}

	centrifugoDashboard = &Centrifugo{
		client: gocent.New(
			gocent.Config{
				Addr:       cfg.CentrifugoURLDashboard,
				Key:        cfg.CentrifugoApiSecretDashboard,
				HTTPClient: httpClient,
			},
		),
	}

	return centrifugoPaymentForm, centrifugoDashboard
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

	return c.client.Publish(ctx, channel, b)
}
