
IMAGE="quay.io/slok/pod-annotate-webhook"

# Build the development docker image
.PHONY: build
build:
	docker build -t $(IMAGE) -f ./Dockerfile ../../

create-certs:
	./create-certs.sh