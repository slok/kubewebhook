## [Unreleased]

### Added

- Add validation allowed Prometheus metrics.

### Changed

- Update to Kubernetes v1.18.

## [0.10.0] - 2020-05-19

### Added

- Dynamic type webhooks without the need to a specific type (can use as multitype webhook).

### Changed

- Fixed on `DELETE` operations static webhooks not receiving object unmarshalled (#41)
- Fixed on `DELETE` operations dynamic webhooks having unmarshaling errors (#63)

## [0.9.1] - 2020-04-09

### Changed

- Update all dependencies including jsonpatch library.

## [0.9.0] - 2020-03-27

### Changed

- Update to Kubernetes v1.18.

## [0.8.0] - 2020-02-18

### Changed

- Update to Kubernetes v1.17.

## [0.7.0] - 2020-02-17

### Changed

- Update to Kubernetes v1.16.

### Fixed

- Use mutation request raw json to create the json patch instead of an unmarshaled object of the raw json. In the
  past we got marshaled the raw into an object, create a deepcopy of the object that would be the mutator, then
  marshal both objects and get the patch.
  This on some cases caused some defaulting on the fields that were not present on the raw json when marshaling/unmarshaling
  process, so when generating the patch the fields that were defaulted acted as if already existed on the original object and
  if modified on the mutated object patch on these field were "modifications" instead of "additions".

## [0.6.0] - 2020-02-16

### Changed

- Update to Kubernetes v1.15.

## [0.5.0] - 2020-02-16

### Changed

- Update to Kubernetes v1.14.

## [0.4.0] - 2020-02-16

### Changed

- Update to Kubernetes v1.13.

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

[unreleased]: https://github.com/slok/kubewebhook/compare/v0.10.0...HEAD
[0.10.0]: https://github.com/slok/kubewebhook/compare/v0.9.1...v0.10.0
[0.9.1]: https://github.com/slok/kubewebhook/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/slok/kubewebhook/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/slok/kubewebhook/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/slok/kubewebhook/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/slok/kubewebhook/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/slok/kubewebhook/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/slok/kubewebhook/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/slok/kubewebhook/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/slok/kubewebhook/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/slok/kubewebhook/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/slok/kubewebhook/releases/tag/v0.1.0
