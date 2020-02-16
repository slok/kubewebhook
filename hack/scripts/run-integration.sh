#!/bin/bash


set -euo pipefail

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TUNNEL_INFO_PATH="/tmp/$(openssl rand -hex 12)-ngrok-tcp-tunnel"
LOCAL_PORT=8080
KUBERNETES_VERSION=v${KUBERNETES_VERSION:-1.15.7}
K3S=${K3S:-false}
PREVIOUS_KUBECTL_CONTEXT=$(kubectl config current-context) || PREVIOUS_KUBECTL_CONTEXT=""

SUDO=''
if [[ $(id -u) -ne 0 ]]; then
    SUDO="sudo"
fi

function cleanup {
    if [ "${K3S}" = true ]; then
        echo "=> Removing K3S cluster"
        $SUDO kill ${K3S_PID}
        $SUDO killall containerd-shim
    else
        echo "=> Removing kind cluster"
        kind delete cluster
        if [ ! -z ${PREVIOUS_KUBECTL_CONTEXT} ]; then 
            echo "=> Setting previous kubectl context"
            kubectl config use-context ${PREVIOUS_KUBECTL_CONTEXT}
        fi
    fi

    echo "=> Removing SSH tunnel"
    kill ${TCP_SSH_TUNNEL_PID}
}
trap cleanup EXIT

# Start Kubernetes cluster.
if [ "${K3S}" = true ]; then
    echo "Start K3S cluster..."
    $SUDO k3s server &
    K3S_PID=$!
    export KUBECONFIG="/etc/rancher/k3s/k3s.yaml"
    $SUDO chmod a+rw ${KUBECONFIG}
else
    echo "Start Kind Kubernetes ${KUBERNETES_VERSION} cluster..."
    kind create cluster --image kindest/node:${KUBERNETES_VERSION}
    export KUBECONFIG="${HOME}/.kube/config"
    chmod a+rw ${KUBECONFIG}
    kubectl config use-context kind-kind
fi

# Sleep a bit so the cluster can start correctly.
echo "Sleeping 30s to give the cluster time to set the runtime..."
sleep 30
    
# Create tunnel.
echo "Start creating SSH tunnel..."
nohup ssh -R 0:localhost:${LOCAL_PORT} tunnel.us.ngrok.com tcp 22 > ${TUNNEL_INFO_PATH} &
sleep 5
TCP_SSH_TUNNEL_PID=$!
TCP_SSH_TUNNEL_ADDRESS=$(cat ${TUNNEL_INFO_PATH} | grep  Forwarding |sed 's/.*tcp:\/\///')
if [[ -z ${TCP_SSH_TUNNEL_ADDRESS} ]]; then
    echo "No TCP address with SSH tunnel, something went wrong, exiting..."
    exit 1
fi
echo "Created tunnel on ${TCP_SSH_TUNNEL_ADDRESS}..."

# Sleep a bit so the cluster can start correctly.
echo "Sleeping 5s to give the SSH tunnel time to connect..."
sleep 5

# Run tests.
echo "Run tests..."
export TEST_WEBHOOK_URL="https://${TCP_SSH_TUNNEL_ADDRESS}"
export TEST_LISTEN_PORT=${LOCAL_PORT}
    ${CURRENT_DIR}/integration-test.sh
