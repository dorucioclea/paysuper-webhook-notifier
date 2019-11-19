module github.com/paysuper/paysuper-webhook-notifier

require (
	github.com/InVisionApp/go-health v2.1.0+incompatible
	github.com/alicebob/gopher-json v0.0.0-20180125190556-5a6b3ba71ee6 // indirect
	github.com/bsm/redis-lock v8.0.0+incompatible
	github.com/centrifugal/gocent v2.0.2+incompatible
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/gogo/protobuf v1.3.0
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/jarcoal/httpmock v1.0.4
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/micro/go-micro v1.8.0
	github.com/micro/go-plugins v1.2.0
	github.com/micro/protobuf v0.0.0-20180321161605-ebd3be6d4fdb
	github.com/paysuper/paysuper-billing-server v0.0.0-20191119092821-15513bedeba8
	github.com/paysuper/paysuper-recurring-repository v1.0.126
	github.com/streadway/amqp v0.0.0-20190827072141-edfb9018d271
	github.com/stretchr/testify v1.4.0
	github.com/yuin/gopher-lua v0.0.0-20190514113301-1cd887cd7036 // indirect
	go.uber.org/zap v1.10.0
	gopkg.in/ProtocolONE/rabbitmq.v1 v1.0.0-20190719062839-9858d727f3ef
)

replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.5.1
