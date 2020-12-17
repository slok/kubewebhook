#!/usr/bin/env bash

set -euxo pipefail

# Add all groups space separated.
GROUPS_VERSION="building:v1"

# Only generate deepcopy (runtime object needs) and typed client.
# Typed listers & informers not required for the moment. Used with generic
# custom informer/listerwatchers.
TARGETS="deepcopy,client"

IMAGE=quay.io/slok/kube-code-generator:v1.19.2
DIR="$( cd "$( dirname "${0}" )" && pwd )"
ROOT_DIR=${DIR}/../..
PROJECT_PACKAGE=github.com/slok/kubewebhook
CRD_PACKAGE=github.com/slok/kubewebhook/test/integration/crd/apis


docker run -it --rm \
        -v ${ROOT_DIR}:/go/src/${PROJECT_PACKAGE} \
        -e PROJECT_PACKAGE="${PROJECT_PACKAGE}" \
        -e CLIENT_GENERATOR_OUT="${PROJECT_PACKAGE}/test/integration/crd/client" \
        -e APIS_ROOT="${CRD_PACKAGE}" \
        -e GROUPS_VERSION="${GROUPS_VERSION}" \
        -e GENERATION_TARGETS="${TARGETS}" \
        ${IMAGE} 

docker run -it --rm \
	-v ${ROOT_DIR}:/src \
        -e GO_PROJECT_ROOT=/src \
	-e CRD_TYPES_PATH=/src/test/integration/crd/apis \
	-e CRD_OUT_PATH=/src/test/integration/crd/manifests \
	${IMAGE} update-crd.sh