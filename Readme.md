<p align="center">
    <img src="logo/kubewebhook_logo@0,5x.png" width="30%" align="center" alt="kubewebhook">
</p>

# kubewebhook [![Build Status][travis-image]][travis-url] [![Go Report Card][goreport-image]][goreport-url] [![GoDoc][godoc-image]][godoc-url]

Kubewebhook is a small Go framework to create [external admission webhooks][aw-url] for Kubernetes.

With Kubewebhook you can make validating and mutating webhooks very fast and focusing mainly on the domain logic of the webhook itself.

## Features

- Ready for mutating and validating webhook kinds.
- Easy and testable API.
- Simple, extensible and flexible.
- Multiple webhooks on the same server.
- Webhook metrics ([RED][red-metrics-url]) for [Prometheus][prometheus-url] with [Grafana dashboard][grafana-dashboard] included.
- Webhook tracing with [Opentracing][opentracing-url].

## Status

Kubewebhook has been used in production for several months, and the results have been good.

## Example

Here is a simple example of mutating webhook that will add `mutated=true` and `mutator=pod-annotate` annotations.

```go
func main() {
    logger := &log.Std{Debug: true}

    cfg := initFlags()

    // Create our mutator
    mt := mutatingwh.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
        pod, ok := obj.(*corev1.Pod)
        if !ok {
            // If not a pod just continue the mutation chain(if there is one) and don't do nothing.
            return false, nil
        }

        // Mutate our object with the required annotations.
        if pod.Annotations == nil {
            pod.Annotations = make(map[string]string)
        }
        pod.Annotations["mutated"] = "true"
        pod.Annotations["mutator"] = "pod-annotate"

        return false, nil
    })

    wh, err := mutatingwh.NewStaticWebhook(mt, &corev1.Pod{}, logger)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
        os.Exit(1)
    }

    // Get the handler for our webhook.
    whHandler := whhttp.HandlerFor(wh)
    logger.Infof("Listening on :8080")
    err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error serving webhook: %s", err)
        os.Exit(1)
    }
}
```

You can get more examples in [here](examples)

## Compatibility matrix

|                  | Kubernetes 1.10 | Kubernetes 1.11 | Kubernetes 1.12 | Kubernetes 1.13 | Kubernetes 1.14 |
| ---------------- | --------------- | --------------- | --------------- | --------------- | --------------- |
| kubewebhook 0.1  | ✓               | ✓               | ?               | ?               | ?               |
| kubewebhook 0.2  | ✓               | ✓               | ?               | ?               | ?               |
| kubewebhook 0.3  | ?               | ?               | ✓               | ?               | ?               |
| kubewebhook HEAD | ?               | ?               | ?               | ✓?              | ?               |

## Documentation

- [Documentation][docs]
- [API][godoc-url]

[travis-image]: https://travis-ci.org/slok/kubewebhook.svg?branch=master
[travis-url]: https://travis-ci.org/slok/kubewebhook
[goreport-image]: https://goreportcard.com/badge/github.com/slok/kubewebhook
[goreport-url]: https://goreportcard.com/report/github.com/slok/kubewebhook
[godoc-image]: https://godoc.org/github.com/slok/kubewebhook?status.svg
[godoc-url]: https://godoc.org/github.com/slok/kubewebhook
[aw-url]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers
[docs]: https://slok.github.io/kubewebhook/
[red-metrics-url]: https://www.weave.works/blog/the-red-method-key-metrics-for-microservices-architecture/
[prometheus-url]: https://prometheus.io/
[grafana-dashboard]: https://grafana.com/dashboards/7088
[opentracing-url]: http://opentracing.io
