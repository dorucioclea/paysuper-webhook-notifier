# PaySuper Webhook Notifier

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-brightgreen.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/paysuper/paysuper-webhook-notifier/issues)
[![Build Status](https://travis-ci.com/paysuper/paysuper-webhook-notifier.svg?branch=develop)](https://travis-ci.com/paysuper/paysuper-webhook-notifier) 
[![codecov](https://codecov.io/gh/paysuper/paysuper-webhook-notifier/branch/develop/graph/badge.svg)](https://codecov.io/gh/paysuper/paysuper-webhook-notifier) 
[![Go Report Card](https://goreportcard.com/badge/github.com/paysuper/paysuper-webhook-notifier)](https://goreportcard.com/report/github.com/paysuper/paysuper-webhook-notifier)

PaySuper Webhook Notifier is a RabbitMQ consumer that sends the PaySuper notifications to projects.

The PaySuper system can send notifications to the project for a set of events during the flow, such as creating new accounts or transaction flow, making payouts, and so on.

In most cases, webhooks are triggered by user actions on your website or by back-end related events like a successful payment, a refund payment and other.

***

## Example of notifications:

1. A request of a payment complete.
2. A request of a refund complete.

[The full list of webhooks' event types.](https://docs.pay.super.com/api/#webhooks)

## Table of Contents

- [Usage](#usage)
- [Contributing](#contributing-feature-requests-and-support)
- [License](#license)

## Usage

The application handles all configuration from the environment variables:

### Environment variables:

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

## Contributing, Feature Requests and Support

If you like this project then you can put a ⭐ on it. It means a lot to us.

If you have an idea of how to improve PaySuper (or any of the product parts) or have general feedback, you're welcome to submit a [feature request](../../issues/new?assignees=&labels=&template=feature_request.md&title=).

Chances are, you like what we have already but you may require a custom integration, a special license or something else big and specific to your needs. We're generally open to such conversations.

If you have a question and can't find the answer yourself, you can [raise an issue](../../issues/new?assignees=&labels=&template=issue--support-request.md&title=I+have+a+question+about+<this+and+that>+%5BSupport%5D) and describe what exactly you're trying to do. We'll do our best to reply in a meaningful time.

We feel that a welcoming community is important and we ask that you follow PaySuper's [Open Source Code of Conduct](https://github.com/paysuper/code-of-conduct/blob/master/README.md) in all interactions with the community.

PaySuper welcomes contributions from anyone and everyone. Please refer to [our contribution guide to learn more](CONTRIBUTING.md).

## License

The project is available as open source under the terms of the [GPL v3 License](https://www.gnu.org/licenses/gpl-3.0).

 
