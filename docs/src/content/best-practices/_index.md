+++
title = "Best practices"
description = ""
weight = 4
alwaysopen = true
+++

# Best practices

When you make a Kubernetes admission webhook there are some points that are need to take into account.

## Latency

The webhooks are called synchronously from the apiserver, this means that when the apiserver starts the chain of admission before storing the object. If the latency of your webhook is high it will impact on every object.

{{% notice tip %}} To reduce the latency if you need resources from the apiserver a good pattern is to use a cache that is populated using a controller that is watching resource changes and populating the cache{{% /notice %}}

## Number of calls

Webhook admissions are requested using TLS, and every webhook configuration can have multiple calls to different webhooks or endpoints. Every call has impact on the chain.

If multiple mutations or validations can be grouped in a single mutation chain that would reduce the latency and the number of calls.

## Controllers and operators

Webhooks should not be used to take other actions on a resource creation, delete or update event. If that's the case a controller or a operator should be used instead. Admission webhooks act on the resource itself before being stored on the ETCD.

## Grouping

If you have custom webhooks on your cluster, would be good to have all the webhooks on the same server and group all mutating webhooks by type.

Example: A server that has 3 webhooks

- Pod mutating webhook
- Pod validating webhook
- Ingress validating webhook.

Each one has a chain with multiple validators and mutators. Example

- Pod mutating webhook.
  - Add team labels based on the namespace.
  - Set tolerations based on the execution type.
  - Set some stablished standard annotations.
  - Add prometheus monitoring port.

That chain would act on one single mutating webhook call and will return all the mutations in one request.

## Namespace

Sometimes the object received for mutating or validating doesn't have the Namespace set, this is mainly because it has not been defaulted yet (AFAIK). In this case you can get the namespace from the admission request, the request is stored on the received context (in the mutators/validators). Check Kubewebhook [context API docs][context-docs]

[context-docs]: https://godoc.org/github.com/slok/kubewebhook/pkg/webhook/context,
