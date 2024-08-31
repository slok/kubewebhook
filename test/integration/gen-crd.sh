#!/usr/bin/env bash

set -euxo pipefail

docker run -it --rm -v ${PWD}:/app ghcr.io/slok/kube-code-generator:v0.3.1 \
        --apis-in ./test/integration/crd/apis \
        --go-gen-out ./test/integration/crd/client \
        --crd-gen-out ./test/integration/crd/manifests
