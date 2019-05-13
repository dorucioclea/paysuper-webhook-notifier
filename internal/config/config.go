package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	CentrifugoUrl string `envconfig:"CENTRIFUGO_URL" required:"true"`
	CentrifugoKey string `envconfig:"CENTRIFUGO_KEY" required:"true"`
	BrokerAddress string `envconfig:"BROKER_ADDRESS" default:"amqp://127.0.0.1:5672"`
	MetricsPort   string `envconfig:"METRICS_PORT" required:"false" default:"8087"`
	MicroRegistry string `envconfig:"MICRO_REGISTRY" required:"false"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)

	return cfg, err
}
