module github.com/slok/kubewebhook

go 1.14

require (
	github.com/opentracing/opentracing-go v1.1.0
	github.com/prometheus/client_golang v1.6.0
	github.com/stretchr/testify v1.5.1
	github.com/uber/jaeger-client-go v2.23.1+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	go.uber.org/atomic v1.6.0 // indirect
	gomodules.xyz/jsonpatch/v3 v3.0.1
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
)
