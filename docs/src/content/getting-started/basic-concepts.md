+++
title = "Basic concepts"
description = ""
weight = 1
alwaysopen = false
+++

# Basic concepts

Kubewebhooks embreces simplicity that's why there are very few components that are required, a webhook server is made up of a server and one or more webhooks.

## Webhooks

There are two types of webhooks, `Validating` and `Mutating`, they are valid [`Webhook`][webhook-docs] interface implementation.

### Validating webhooks

These webhooks only validate the received object, a Validating webhook is made up of a [`Validator`][validator-docs] interface, this interface only validates an `metav1.Object`.

Validating webhooks are with [`admissionregistration.k8s.io/v1beta1/ValidatingWebhookConfiguration`][validatingwebhook-k8s-api] Kubernetes resource

{{% alert theme="success" %}}A `Validator` can act also as a validator chain, Check [`validating.Chain`](https://godoc.org/github.com/slok/kubewebhook/pkg/webhook/validating#Chain){{% /alert %}}

### Mutating webhooks

Mutating webhooks are similar to validating webhooks but instead of validating the object it modifies (mutate) them. A Mutating webhook is made up of [`Mutator`][mutator-docs] interface.

Mutating webhooks are with [`admissionregistration.k8s.io/v1beta1/MutatingWebhookConfiguration`][mutatingwebhook-k8s-api] Kubernetes resource

{{% alert theme="success" %}}A `Mutator` can act also as a mutator chain, Check [`mutating.Chain`](https://godoc.org/github.com/slok/kubewebhook/pkg/webhook/mutating#Chain){{% /alert %}}

## HTTP Handler

Kubewebhook focuses on the Kubernetes webhooks itself, it doesn't try to manage the http server, that's why it will provide a Go [`http.Handler`][http-handler] and you can set this handler on the [`http.Server`][http-server] that you want with the options.

This approach makes gives the user the flexibility to serve the webhook in very customized ways like number of connections, address, paths...

To get a Handler from a previous created Webhook you can use [`http.HandlerFor`][handlerfor-docs] method.

{{% alert theme="success" %}}You can create a single HTTP server with multiple webhooks (multiple hadlers){{% /alert %}}

## Context

Every webhook receives and passes a `context.Context`. In this context is also stored the orignal `admissionv1beta1.AdmissionRequest` in case more information is required in a mutator or a validator, like the operation of the webhook (`CREATE`, `UPDATE`...).

[webhook-docs]: https://godoc.org/github.com/slok/kubewebhook/pkg/webhook#Webhook
[validator-docs]: https://godoc.org/github.com/slok/kubewebhook/pkg/webhook/validating#Validator
[mutator-docs]: https://godoc.org/github.com/slok/kubewebhook/pkg/webhook/mutating#Mutator
[validatingwebhook-k8s-api]: https://v1-10.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#validatingwebhookconfiguration-v1beta1-admissionregistration-k8s-io
[mutatingwebhook-k8s-api]: https://v1-10.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#mutatingwebhookconfiguration-v1beta1-admissionregistration-k8s-io
[http-handler]: https://golang.org/pkg/net/http/#Handler
[http-server]: https://golang.org/pkg/net/http/#Server
[handlerfor-docs]: https://github.com/slok/kubewebhook/blob/master/pkg/http/handler.go#L24
