
## [Unreleased]

## [0.3.0] - 2019-05-30
### Added
- Util to know if a admission review is dry run.

### Changed
- Update to Kubernetes v1.12.

## [0.2.0] - 2018-09-29

Breaking: Webhook constructors now need a tracer.

### Added
- Open tracing support on validators.
- Open tracing support on mutators.
- Open tracing support on webhooks.

## [0.1.1] - 2018-07-22
### Added
- MustHandlerFor in case don't want to get an error (panic instead) and be less verbose.

### Fixed
- Set internal server error status code (500) when a error on a webhook happens.

## [0.1.0] - 2018-07-22
### Added
- Grafana dashboard for Prometheus metrics.
- Webhook admission review Prometheus metrics.
- Pass admission request on the context to the webhooks.
- Pass request context to the webhook and its mutating/validating chain.
- Static validating webhook.
- Mutating webhook example.
- Static mutating webhook.
- Handler creator for webhooks.

[Unreleased]: https://github.com/slok/kubewebhook/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/slok/kubewebhook/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/slok/kubewebhook/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/slok/kubewebhook/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/slok/kubewebhook/releases/tag/v0.1.0