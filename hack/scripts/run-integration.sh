#!/bin/bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DOMAIN_PREFIX=$(openssl rand -hex 12)
DOMAIN=${DOMAIN_PREFIX}.serveo.net
EXPOSED_PORT=$(shuf -i 1500-35000 -n 1)
LOCAL_PORT=8080
KUBERNETES_VERSION=v${KUBERNETES_VERSION:-1.13.6}
#K3S=true

SUDO=''
if [[ $(id -u) -ne 0 ]]; then
    SUDO="sudo"
fi

function cleanup {
    if [[ ! -z ${K3S} ]]; then
        echo "=> Removing K3S cluster"
        $SUDO kill ${K3S_PID}
        $SUDO killall containerd-shim
    else
        echo "=> Removing kind cluster"
        $SUDO kind delete cluster
    fi

    $SUDO kill ${SSH_TUNNEL_PID}
}
trap cleanup EXIT

# Start Kubernetes cluster.
if [[ ! -z ${K3S} ]]; then
    echo "Start K3S cluster..."
    $SUDO k3s server &
    K3S_PID=$!
    export KUBECONFIG="/etc/rancher/k3s/k3s.yaml"
else
    echo "Start Kind Kubernetes ${KUBERNETES_VERSION} cluster..."
    $SUDO kind create cluster --image kindest/node:${KUBERNETES_VERSION}
    export KUBECONFIG="$(kind get kubeconfig-path)"
fi

$SUDO chmod a+r ${KUBECONFIG}

# Sleep a bit so the cluster can start correctly.
echo "Sleeping 20s to give the cluster time to set the runtime..."
sleep 20

# Create tunnel.
echo "Create tunnel on ${DOMAIN}:${EXPOSED_PORT}..."
ssh -R ${DOMAIN_PREFIX}:${EXPOSED_PORT}:localhost:${LOCAL_PORT} serveo.net &
SSH_TUNNEL_PID=$!

# Sleep a bit so the cluster can start correctly.
echo "Sleeping 5s to give the SSH tunnel time to connect..."
sleep 5


# Run tests.
echo "Run tests..."
export TEST_WEBHOOK_URL="https://${DOMAIN}:${EXPOSED_PORT}"
export TEST_LISTEN_PORT=${LOCAL_PORT}
${CURRENT_DIR}/integration-test.sh
