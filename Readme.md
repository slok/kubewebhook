<p align="center">
    <img src="logo/kubewebhook_logo@0,5x.png" width="30%" align="center" alt="kubewebhook">
</p>

# kubewebhook [![Build Status][ci-image]][ci-url] [![Go Report Card][goreport-image]][goreport-url] [![GoDoc][godoc-image]][godoc-url]

Kubewebhook is a small Go framework to create [external admission webhooks][aw-url] for Kubernetes.

With Kubewebhook you can make validating and mutating webhooks in any version, fast, easy, and focusing mainly on the domain logic of the webhook itself.

## Features

- Ready for mutating and validating webhook kinds.
- Abstracts webhook versioning (compatible with `v1beta1` and `v1`).
- Resource inference (compatible with `CRD`s and fallbacks to [`Unstructured`][runtime-unstructured]).
- Easy and testable API.
- Simple, extensible and flexible.
- Multiple webhooks on the same server.
- Webhook metrics ([RED][red-metrics-url]) for [Prometheus][prometheus-url] with [Grafana dashboard][grafana-dashboard] included.
- Supports [warnings].

## Example

```go
func run() error {
    logger := &kwhlog.Std{Debug: true}

    // Create our mutator
    mt := kwhmutating.MutatorFunc(func(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
        pod, ok := obj.(*corev1.Pod)
        if !ok {
            return &kwhmutating.MutatorResult{}, nil
        }

        // Mutate our object with the required annotations.
        if pod.Annotations == nil {
            pod.Annotations = make(map[string]string)
        }
        pod.Annotations["mutated"] = "true"
        pod.Annotations["mutator"] = "pod-annotate"

        return &kwhmutating.MutatorResult{MutatedObject: pod}, nil
    })

    // Create webhook.
    mcfg := kwhmutating.WebhookConfig{
        ID:      "pod-annotate",
        Mutator: mt,
        Logger:  logger,
    }
    wh, err := kwhmutating.NewWebhook(mcfg)
    if err != nil {
        return fmt.Errorf("error creating webhook: %w", err)
    }

    // Get HTTP handler from webhook.
    whHandler, err := kwhhttp.HandlerFor(wh)
    if err != nil {
        return fmt.Errorf("error creating webhook handler: %w", err)
    }

    // Serve.
    logger.Infof("Listening on :8080")
    err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
    if err != nil {
        return fmt.Errorf("error serving webhook: %w", err)
    }

    return nil
```

You can get more examples in [here](examples)

## Production ready example

This repository is a production ready webhook app: https://github.com/slok/k8s-webhook-example

It shows, different webhook use cases, app structure, testing domain logic, kubewebhook use case, how to deploy...

## Static and dynamic webhooks

We have 2 kinds of webhooks:

- Static: Common one, is a single resource type webhook.
  - Use [`mutating.WebhookConfig.Obj`][mutating-cfg] to configure.
  - Use [`validating.WebhookConfig.Obj`][validating-cfg] to configure.
- Dynamic: Used when the same webhook act on multiple types, unknown types and/or is used for generic stuff (e.g labels).
  - To use this kind of webhook, don't set the type on the configuration or set to `nil`.
  - If a request for an unknown type is not known by the webhook libraries, it will fallback to [`runtime.Unstructured`][runtime-unstructured] object type.
  - Very useful to manipulate multiple resources on the same webhook (e.g `Deployments`, `Statfulsets`).
  - CRDs are unknown types so they will fallback to [`runtime.Unstructured`][runtime-unstructured]`.
  - If using CRDs, better use `Static` webhooks.
  - Very useful to maniputale any `metadata` based validation or mutations (e.g `Labels, annotations...`)

## Compatibility matrix

The Kubernetes' version associated with Kubewebhook's versions means that this specific version
is tested and supports the shown K8s version, however, this doesn't mean that doesn't work with other versions. Normally they work with multiple versions (e.g `v1.18` and `v1.19`).

| Kubewebhook version | k8s version | Supported admission reviews | Support dynamic webhooks |
| ------------------- | ----------- | --------------------------- | ------------------------ |
| v2.0                | 1.19        | v1beta1, v1                 | ✔                        |
| v0.11               | 1.19        | v1beta1                     | ✔                        |
| v0.10               | 1.18        | v1beta1                     | ✔                        |
| v0.9                | 1.18        | v1beta1                     | ✖                        |
| v0.8                | 1.17        | v1beta1                     | ✖                        |
| v0.7                | 1.16        | v1beta1                     | ✖                        |
| v0.6                | 1.15        | v1beta1                     | ✖                        |
| v0.5                | 1.14        | v1beta1                     | ✖                        |
| v0.4                | 1.13        | v1beta1                     | ✖                        |
| v0.3                | 1.12        | v1beta1                     | ✖                        |
| v0.2                | 1.11        | v1beta1                     | ✖                        |
| v0.2                | 1.10        | v1beta1                     | ✖                        |

## Documentation

You can access [here][godoc-url].

[ci-image]: https://github.com/slok/kubewebhook/workflows/CI/badge.svg
[ci-url]: https://github.com/slok/kubewebhook/actions
[goreport-image]: https://goreportcard.com/badge/github.com/slok/kubewebhook
[goreport-url]: https://goreportcard.com/report/github.com/slok/kubewebhook
[godoc-image]: https://godoc.org/github.com/slok/kubewebhook?status.svg
[godoc-url]: https://pkg.go.dev/github.com/slok/kubewebhook?tab=doc
[aw-url]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers
[docs]: https://slok.github.io/kubewebhook/
[red-metrics-url]: https://www.weave.works/blog/the-red-method-key-metrics-for-microservices-architecture/
[prometheus-url]: https://prometheus.io/
[grafana-dashboard]: https://grafana.com/dashboards/7088
[mutating-cfg]: https://pkg.go.dev/github.com/slok/kubewebhook/pkg/webhook/mutating?tab=doc#WebhookConfig
[validating-cfg]: https://pkg.go.dev/github.com/slok/kubewebhook/pkg/webhook/validating?tab=doc#WebhookConfig
[runtime-unstructured]: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime?tab=doc#Unstructured
[warnings]: https://kubernetes.io/blog/2020/09/03/warnings/
