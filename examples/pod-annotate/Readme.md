# pod-annotate

This example is a simple mutating webhook that adds annotations.

## steps

### Set up the mutating webhook

- Webhooks need TLS, so we need to create certificates using `make create-certs`.
- Deploy the mutating webhook certificates: `kubectl apply -f ./webhook-certs.yaml`.
- Deploy the mutating webhook: `kubectl apply -f ./webhook.yaml`.
- Register the mutating webhook for the apiserver: `kubectl apply -f ./webhook-registration.yaml`.

### Check

- Deploy a test with: `kubectl apply -f ./test-deployment.yaml`.
- Check all the annotations of the created pods, for example with `kubect get pods -o yaml`
