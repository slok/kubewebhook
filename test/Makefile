# Creates test certificates for the application.
.PHONY: create-manual-test-certs
create-manual-test-certs:
	${PWD}/create-manual-certs.sh 0.tcp.ngrok.io ${PWD}/manual/certs

.PHONY: run-example
run-example:
	go run ${PWD}/../examples/multiwebhook/cmd/multiwebhook/* -tls-cert-file=${PWD}/manual/certs/cert.pem -tls-key-file=${PWD}/manual/certs/key.pem --debug
