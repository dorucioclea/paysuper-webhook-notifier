WebHook Notification Service
====

[![Build Status](https://travis-ci.org/paysuper/paysuper-webhook-notifier.svg?branch=master)](https://travis-ci.org/paysuper/paysuper-webhook-notifier) 
[![codecov](https://codecov.io/gh/paysuper/paysuper-webhook-notifier/branch/master/graph/badge.svg)](https://codecov.io/gh/paysuper/paysuper-webhook-notifier) [![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=paysuper_paysuper-webhook-notifier&metric=alert_status)](https://sonarcloud.io/dashboard?id=paysuper_paysuper-webhook-notifier)

This service is rabbitmq consumer for sending other notification from PaySuper to projects.
Example of sending notifications:

1. Payment complete request
2. Refund complete request
3. User account validation
4. And other

## Environment variables:

| Name                     | Required | Default               | Description                                                                                                                         |
|:-------------------------|:--------:|:----------------------|:---------------------------------------------------------------------------------------------------------------------|
| CENTRIFUGO_URL           | true     | -                     | Centrifugo host address                                                                                              |
| CENTRIFUGO_KEY           | true     | -                     | Centrifugo API key                                                                                                   |
| CENTRIFUGO_USER_CHANNEL  | -        | paysuper:order#%s     | Name of centrifugo channel to send notifications to users. Placeholder in the end will to change to order identifier |
| CENTRIFUGO_ADMIN_CHANNEL | -        | paysuper:admin        | Name of centrifugo channel to send notifications to administrators                                                   |
| BROKER_ADDRESS           | -        | amqp://127.0.0.1:5672 | RabbitMQ url address                                                                                                 |
| METRICS_PORT             | -        | 8087                  | Http server port for health and metrics request                                                                      |
| REDIS_HOST               | -        | 127.0.0.1:6379        | Redis server host address                                                                                            |
| REDIS_PASSWORD           | -        | ""                    | Password to access to Redis server                                                                                   |
