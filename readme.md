WebHook Notification Service
====

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-brightgreen.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Build Status](https://travis-ci.org/paysuper/paysuper-webhook-notifier.svg?branch=master)](https://travis-ci.org/paysuper/paysuper-webhook-notifier) 
[![codecov](https://codecov.io/gh/paysuper/paysuper-webhook-notifier/branch/master/graph/badge.svg)](https://codecov.io/gh/paysuper/paysuper-webhook-notifier) 
[![Go Report Card](https://goreportcard.com/badge/github.com/paysuper/paysuper-webhook-notifier)](https://goreportcard.com/report/github.com/paysuper/paysuper-webhook-notifier)

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

## Contributing
We feel that a welcoming community is important and we ask that you follow PaySuper's [Open Source Code of Conduct](https://github.com/paysuper/code-of-conduct/blob/master/README.md) in all interactions with the community.

PaySuper welcomes contributions from anyone and everyone. Please refer to each project's style and contribution guidelines for submitting patches and additions. In general, we follow the "fork-and-pull" Git workflow.

The master branch of this repository contains the latest stable release of this component.

 
