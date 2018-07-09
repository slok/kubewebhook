+++
title = "Kubewebhook documentation"
description = ""
+++

![logo](/img/logo.png)

# Kubewebhook

Kubewebhook is a small Go framework to create [external admission webhooks][aw-url] for Kubernetes.

With Kubewebhook you can make validating and mutating webhooks very fast and focusing mainly on the domain logic of the webhook itself.

## Main features

- Ready for mutating and validating webhook kinds.
- Easy and testable API.
- Simple, extensible and flexible.
- Multiple webhooks on the same server.

[aw-url]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers
