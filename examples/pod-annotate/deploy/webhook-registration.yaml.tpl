apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: pod-annotate-webhook
  labels:
    app: pod-annotate-webhook
    kind: mutator
webhooks:
  - name: pod-annotate-webhook.slok.xyz
    clientConfig:
      service:
        name: pod-annotate-webhook
        namespace: default
        path: "/mutate"
      caBundle: CA_BUNDLE
    rules:
      - operations: [ "CREATE" ]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
        