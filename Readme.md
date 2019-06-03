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

Integration tests will run on different Kubernetes versions, so if these are passing is likely that HEAD supports those Kubernetes versions, these will be marked on the matrix as `✓?`. Check the latest builds [here][travis-url]

|                  | Kubernetes 1.10 | Kubernetes 1.11 | Kubernetes 1.12 | Kubernetes 1.13 | Kubernetes 1.14 |
| ---------------- | --------------- | --------------- | --------------- | --------------- | --------------- |
| kubewebhook 0.1  | ✓               | ✓               | ?               | ?               | ?               |
| kubewebhook 0.2  | ✓               | ✓               | ?               | ?               | ?               |
| kubewebhook 0.3  | ?               | ?               | ✓               | ?               | ?               |
| kubewebhook HEAD | ?               | ?               | ✓?              | ✓?              | ✓?              |

## Documentation

- [Documentation][docs]
- [API][godoc-url]

## Integration tests

Tools required

- [mkcert] (optional if you want to create new certificates).
- [kind] (option1, to run the cluster).
- [k3s] (option2, to run the cluster)
- ssh (to expose our webhook to the internet).

### (Optional) Certificates

Certificates are ready to be used on [/test/integration/certs]. This certificates are valid for `*.serveo.net` so, they should be valid for our exposed webhooks using [serveo].

If you want to create new certificates execute this:

```bash
make create-integration-test-certs
```

### Running the tests

The integration tests are on [/tests/integration], there are the certificates valid for `serveo.net` where the tunnel will be exposing the webhooks.

Go integration tests require this env vars:

- `TEST_WEBHOOK_URL`: The url where the apiserver should make the webhook requests.
- `TEST_LISTEN_PORT`: The port where our webhook will be listening the requests.

There are 2 ways of bootstrapping the integration tests, one using kind and another using [k3s].

To run the integration tests do:

```bash
make integration-test
```

This it will bootstrap a cluster with [kind] by default and a [k3s] cluster if `K3S=true` env var is set. A ssh tunnel in a random subdomain using the 1987 port, and finally use the precreated certificates (see previous step), after this will execute the tests, and do it's best effort to tear down the clusters (on k3s could be problems, so have a check on k3s processes).

### Developing integration tests

To develop integration test is handy to run a k3s cluster and a serveo tunnel, then check out [/tests/integration/helper/config] and use this development settings on the integration tests.

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
[mkcert]: https://github.com/FiloSottile/mkcert
[kind]: https://github.com/kubernetes-sigs/kind
[k3s]: https://k3s.io
[serveo]: https://serveo.net
