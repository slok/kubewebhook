apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: ingress-host-validator-webhook
  labels:
    app: ingress-host-validator-webhook
    kind: validating
webhooks:
  - name: ingress-host-validator-webhook.slok.xyz
    clientConfig:
      service:
        name: ingress-host-validator-webhook
        namespace: default
        path: "/validating"
      caBundle: CA_BUNDLE
    rules:
      - operations: [ "CREATE", "UPDATE" ]
        apiGroups: ["extensions"]
        apiVersions: ["v1beta1"]
        resources: ["ingresses"]
        