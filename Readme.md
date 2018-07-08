<p align="center">
    <img src="logo/kubewebhook_logo@0,5x.png" width="50%" align="center" alt="kubewebhook">
</p>

# kubewebhook

Kubewebhook is a small Go framework to create [external admission webhooks][aw-url] for Kubernetes.

With Kubewebhook you can make validating and mutating webhooks very fast and focusing mainly on the domain logic of the webhook itself.

## Features

- Ready for mutating and validating webhook kinds.
- Easy and testable API.
- Simple, extensible and flexible.
- Multiple webhooks on the same server.

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

[aw-url]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers
