package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	CentrifugoUrl string `envconfig:"CENTRIFUGO_URL" required:"true"`
	CentrifugoKey string `envconfig:"CENTRIFUGO_KEY" required:"true"`
	BrokerAddress string `envconfig:"BROKER_ADDRESS" default:"amqp://127.0.0.1:5672"`
	MetricsPort   string `envconfig:"METRICS_PORT" required:"false" default:"8087"`
	RedisHost     string `envconfig:"REDIS_HOST" default:"127.0.0.1:6379"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`

	CentrifugoUserChannel            string `envconfig:"CENTRIFUGO_USER_CHANNEL" default:"paysuper:order#%s"`
	CentrifugoAdminChannel           string `envconfig:"CENTRIFUGO_ADMIN_CHANNEL" default:"paysuper:admin"`
	CentrifugoMerchantTestingChannel string `envconfig:"CENTRIFUGO_MERCHANT_CHANNEL" default:"paysuper:merchant:order_testing#%s"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)

	return cfg, err
}
