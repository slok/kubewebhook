#!/usr/bin/env bash

set -euxo pipefail

# Add all groups space separated.
GROUPS_VERSION="building:v1"

# Only generate deepcopy (runtime object needs) and typed client.
# Typed listers & informers not required for the moment. Used with generic
# custom informer/listerwatchers.
TARGETS="deepcopy,client"

IMAGE=quay.io/slok/kube-code-generator:v1.21.0
DIR="$( cd "$( dirname "${0}" )" && pwd )"
ROOT_DIR=${DIR}/../..
PROJECT_PACKAGE=github.com/slok/kubewebhook/v2
CRD_PACKAGE=github.com/slok/kubewebhook/v2/test/integration/crd/apis


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

# Kubewebhook imports are v2, but not its path, the easies way for not having problems on the
# generated code is to generate as a regular v1 (not imporot change), and then we replace it
# the imports. Kind of hackish, but easier than dealing with all the import problems on the CRD
# generating tools.

# With the first replace we ensure that all the v2 (shouldn't be any, but just in case), are set without v2,
# and then we replace the v1 imports with v2 imports
#       find "${ROOT_DIR}/test/integration/" -iname *.go -type f -exec sed -i -e 's/\"github.com\/slok\/kubewebhook\/v2/\"github.com\/slok\/kubewebhook/g' {} \;
#       find "${ROOT_DIR}/test/integration/" -iname *.go -type f -exec sed -i -e 's/\"github.com\/slok\/kubewebhook/\"github.com\/slok\/kubewebhook\/v2/g' {} \;