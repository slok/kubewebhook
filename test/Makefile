CREATE_MANUAL_TEST_CERTS_CMD := ${PWD}/create-manual-certs.sh 0.tcp.ngrok.io ${PWD}/manual/certs
RUN_EXAMPLE_CMD := go run ${PWD}/../examples/multiwebhook/cmd/multiwebhook/* -tls-cert-file=${PWD}/manual/certs/cert.pem -tls-key-file=${PWD}/manual/certs/key.pem --debug

# Creates test certificates for the application.
.PHONY: create-manual-test-certs
create-manual-test-certs:
	$(CREATE_MANUAL_TEST_CERTS_CMD)

.PHONY: run-example
run-example:
	$(RUN_EXAMPLE_CMD)
	