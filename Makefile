
# Name of this service/application
SERVICE_NAME := kubewebhook

# Path of the go service inside docker
DOCKER_GO_SERVICE_PATH := /go/src/github.com/slok/kubewebhook

# Shell to use for running scripts
SHELL := $(shell which bash)

# Get docker path or an empty string
DOCKER := $(shell command -v docker)

# Get the main unix group for the user running make (to be used by docker-compose later)
GID := $(shell id -g)

# Get the unix user id for the user running make (to be used by docker-compose later)
UID := $(shell id -u)

# Commit hash from git
COMMIT=$(shell git rev-parse --short HEAD)

# cmds
UNIT_TEST_CMD := ./hack/scripts/unit-test.sh
INTEGRATION_TEST_CMD := ./hack/scripts/integration-test.sh
MOCKS_CMD := ./hack/scripts/mockgen.sh
DOCKER_RUN_CMD := docker run -v ${PWD}:$(DOCKER_GO_SERVICE_PATH) --rm -it $(SERVICE_NAME)
DEP_ENSURE_CMD := dep ensure

# environment dirs
DEV_DIR := docker/dev

# The default action of this Makefile is to build the development docker image
.PHONY: default
default: build

# Test if the dependencies we need to run this Makefile are installed
.PHONY: deps-development
deps-development:
ifndef DOCKER
	@echo "Docker is not available. Please install docker"
	@exit 1
endif

# Build the development docker image
.PHONY: build
build:
	docker build -t $(SERVICE_NAME) --build-arg uid=$(UID) --build-arg  gid=$(GID) -f ./docker/dev/Dockerfile .

# Shell the development docker image
.PHONY: build
shell: build
	$(DOCKER_RUN_CMD) /bin/bash

# Build production stuff.
build-binary:
	$(DOCKER_RUN_CMD) /bin/sh -c '$(BUILD_BINARY_CMD)'

.PHONY: build-image
build-image:
	$(BUILD_IMAGE_CMD)

# Test stuff in dev
.PHONY: unit-test
unit-test: build
	$(DOCKER_RUN_CMD) /bin/sh -c '$(UNIT_TEST_CMD)'
.PHONY: integration-test
integration-test: build
	$(DOCKER_RUN_CMD) /bin/sh -c '$(INTEGRATION_TEST_CMD)'
.PHONY: test
test: integration-test

# Test stuff in ci
.PHONY: ci-unit-test
ci-unit-test:
	$(UNIT_TEST_CMD)
.PHONY: ci-integration-test
ci-integration-test:
	$(INTEGRATION_TEST_CMD)
.PHONY: ci
ci: ci-integration-test

# Mocks stuff in dev
.PHONY: mocks
mocks: build
	$(DOCKER_RUN_CMD) /bin/sh -c '$(MOCKS_CMD)'