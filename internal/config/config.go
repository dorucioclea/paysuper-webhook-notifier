package config

import "github.com/kelseyhightower/envconfig"

type Centrifugo struct {
	CentrifugoApiSecretPaymentForm string `envconfig:"CENTRIFUGO_API_SECRET_PAYMENT_FORM" required:"true"`
	CentrifugoURLPaymentForm       string `envconfig:"CENTRIFUGO_URL_PAYMENT_FORM" required:"true"`
	CentrifugoApiSecretDashboard   string `envconfig:"CENTRIFUGO_API_SECRET_DASHBOARD" required:"true"`
	CentrifugoURLDashboard         string `envconfig:"CENTRIFUGO_URL_DASHBOARD" required:"true"`
	CentrifugoUserChannel          string `envconfig:"CENTRIFUGO_USER_CHANNEL" default:"paysuper:order#%s"`
	CentrifugoAdminChannel         string `envconfig:"CENTRIFUGO_ADMIN_CHANNEL" default:"paysuper:admin"`
}

type Config struct {
	BrokerAddress string `envconfig:"BROKER_ADDRESS" default:"amqp://127.0.0.1:5672"`
	MetricsPort   string `envconfig:"METRICS_PORT" required:"false" default:"8087"`
	RedisHost     string `envconfig:"REDIS_HOST" default:"127.0.0.1:6379"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`

	*Centrifugo
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)

	return cfg, err
}
