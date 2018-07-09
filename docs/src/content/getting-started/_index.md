+++
title = "Getting started"
description = ""
weight = 1
alwaysopen = true
+++

# Getting started

If it's the first time that you are using Kubewebhook you should get familiar with the few pieces that are needed to create a webhook.

- [basic concepts][basic-concepts]

Now you are ready to get stared with with Kubewebook, you can choose to start with a mutating webhook or a validating webhook, in the end both use the same approach, so it doesn't matter.

- Mutating tutorial (TODO, [check example for now][mutating-example])
- Validating tutorial (TODO, [check example for now][validating-example])

[basic-concepts]: {{< relref "getting-started/basic-concepts.md" >}}
[mutating-example]: https://github.com/slok/kubewebhook/tree/master/examples/pod-annotate
[validating-example]: https://github.com/slok/kubewebhook/tree/master/examples/ingress-host-validator
