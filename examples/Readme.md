# Examples

## pod-annotate

This example is a simple mutating webhook that adds annotations.

### steps

#### Set up the mutating webhook

- Deploy the mutating webhook certificates: `kubectl apply -f ./pod-annotate/deploy/webhook-certs.yaml`.
- Deploy the mutating webhook: `kubectl apply -f ./pod-annotate/deploy/webhook.yaml`.
- Register the mutating webhook for the apiserver: `kubectl apply -f ./pod-annotate/deploy/webhook-registration.yaml`.

#### Check

- Deploy a test with: `kubectl apply -f ./pod-annotate/deploy/test-deployment.yaml`.
- Check all the annotations of the created pods, for example with `kubect get pods -o yaml`

## ingress-host-validator

This example validates that all the ingress rules host match a regex, if it doesn't match it will not admit that ingress,

### steps

#### Set up the mutating webhook

- Deploy the validating webhook certificates: `kubectl apply -f ./ingress-host-validator/deploy/webhook-certs.yaml`.
- Deploy the validating webhook: `kubectl apply -f ./ingress-host-validator/deploy/webhook.yaml`.
- Register the validating webhook for the apiserver: `kubectl apply -f ./ingress-host-validator/deploy/webhook-registration.yaml`.

#### Check

- Deploy a test with: `kubectl apply -f ./test-ingress.yaml` and watch the denied and accepted ingresses.
