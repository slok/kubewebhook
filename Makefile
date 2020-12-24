
# Name of this service/application
SERVICE_NAME := kubewebhook
SHELL := $(shell which bash)
GID := $(shell id -g)
UID := $(shell id -u)
COMMIT=$(shell git rev-parse --short HEAD)

# cmds
UNIT_TEST_CMD := ./hack/scripts/unit-test.sh
INTEGRATION_TEST_CMD := ./hack/scripts/run-integration.sh
MOCKS_CMD := ./hack/scripts/mockgen.sh
DOCKER_RUN_CMD := docker run -v ${PWD}:/src --rm -it $(SERVICE_NAME)
DEPS_CMD := go mod tidy
CHECK_CMD := ./hack/scripts/check.sh

help: ## Show this help.
	@echo "Help"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-20s\033[93m %s\n", $$1, $$2}'

.PHONY: default
default: help

.PHONY: build
build: ## Build the development docker images.
	docker build -t $(SERVICE_NAME) --build-arg uid=$(UID) --build-arg  gid=$(GID) -f ./docker/dev/Dockerfile .
	docker build -t $(SERVICE_NAME)-docs --build-arg uid=$(UID) --build-arg  gid=$(GID) -f ./docker/docs/Dockerfile .

build-binary: ## Build production stuff.
	$(DOCKER_RUN_CMD) /bin/sh -c '$(BUILD_BINARY_CMD)'

.PHONY: build-image
build-image: ## Build docker image.
	$(BUILD_IMAGE_CMD)

.PHONY: unit-test
unit-test: build ## Execute unit tests.
	$(DOCKER_RUN_CMD) /bin/sh -c '$(UNIT_TEST_CMD)'

.PHONY: integration-test
integration-test: build ## Execute integration tests.
	$(INTEGRATION_TEST_CMD)

.PHONY: test ## Alias for unit-test
test: unit-test

.PHONY: check
check: build ## Runs checks.
	@$(DOCKER_RUN_CMD) /bin/sh -c '$(CHECK_CMD)'

.PHONY: ci-unit-test
ci-unit-test: ## Same as unit-test but for CI.
	$(UNIT_TEST_CMD)

.PHONY: ci-integration-test
ci-integration-test: ## Same as integration-tests but for CI.
	$(INTEGRATION_TEST_CMD)

.PHONY: ci  ## Execute all the tests for CI.
ci: ci-unit-test ci-integration-test

.PHONY: mocks
mocks: build ## Generate mocks.
	$(DOCKER_RUN_CMD) /bin/sh -c '$(MOCKS_CMD)'

.PHONY: godoc
godoc: ## Run library docs.
	godoc -http=":6060"

.PHONY: deps
deps: ## Setup dependencies.
	$(DEPS_CMD)

.PHONY: create-integration-test-certs
integration-create-certs: ## Creates certificates for the integration test.
	./test/integration/create-certs.sh

.PHONY: generate-integration-test-crd
integration-gen-crd: ## Generates CRDs for the integration test.
	./test/integration/gen-crd.sh
