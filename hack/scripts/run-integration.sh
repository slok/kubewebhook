#!/bin/bash


set -euo pipefail

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TUNNEL_INFO_PATH="/tmp/$(openssl rand -hex 12)-ngrok-tcp-tunnel"
LOCAL_PORT=8080
KUBERNETES_VERSION=v${KUBERNETES_VERSION:-1.17.0}
PREVIOUS_KUBECTL_CONTEXT=$(kubectl config current-context) || PREVIOUS_KUBECTL_CONTEXT=""
# If NGROK used, we need ngrok key in b64 set in `NGROK_SSH_PRIVATE_KEY_B64`:
#   - echo -e "Host tunnel.us.ngrok.com\n\tStrictHostKeyChecking no\n" >> ~/.ssh/config
#   - echo -e ${NGROK_SSH_PRIVATE_KEY_B64} | base64 -d > ~/.ssh/id_ed25519
#   - chmod 400 ~/.ssh/id_ed25519
# If serveo used (Unlimited tunnels):
#   - echo -e "Host serveo.net\n\tStrictHostKeyChecking no\n" >> ~/.ssh/config
NGROK=${NGROK:-false}

function cleanup {
    echo "=> Removing kind cluster"
    kind delete cluster
    if [ ! -z ${PREVIOUS_KUBECTL_CONTEXT} ]; then 
        echo "=> Setting previous kubectl context"
        kubectl config use-context ${PREVIOUS_KUBECTL_CONTEXT}
    fi

    echo "=> Removing SSH tunnel"
    kill ${TCP_SSH_TUNNEL_PID}
}
trap cleanup EXIT

# Start Kubernetes cluster.
echo "Start Kind Kubernetes ${KUBERNETES_VERSION} cluster..."
kind create cluster --image kindest/node:${KUBERNETES_VERSION}
export KUBECONFIG="${HOME}/.kube/config"
chmod a+rw ${KUBECONFIG}
kubectl config use-context kind-kind

# Sleep a bit so the cluster can start correctly.
echo "Sleeping 30s to give the cluster time to set the runtime..."
sleep 30
    
# Create tunnel.
echo "Start creating SSH tunnel..."
if [ "${NGROK}" = true ]; then
    echo "Start NGROK tunnel..."
    nohup ssh -R 0:localhost:${LOCAL_PORT} tunnel.us.ngrok.com tcp 22 > ${TUNNEL_INFO_PATH} &
    sleep 5
    TCP_SSH_TUNNEL_PID=$!
    TCP_SSH_TUNNEL_ADDRESS=$(cat ${TUNNEL_INFO_PATH} | grep  Forwarding |sed 's/.*tcp:\/\///')
else
    echo "Start serveo tunnel..."
    # Force pseudo terminal.
    nohup ssh -tt -R 0:localhost:${LOCAL_PORT} serveo.net > ${TUNNEL_INFO_PATH} &
    sleep 5
    TCP_SSH_TUNNEL_PID=$!
    # Filter port from got string  (using sed) and sanitize to remove all non digits (using tr).
    TCP_SSH_TUNNEL_PORT=$(cat ${TUNNEL_INFO_PATH} | grep  Forwarding | sed 's/.*net://' | tr -dc '[:digit:]')
    TCP_SSH_TUNNEL_ADDRESS="serveo.net:${TCP_SSH_TUNNEL_PORT}"
fi

if [[ -z ${TCP_SSH_TUNNEL_ADDRESS} ]]; then
    echo "No TCP address with SSH tunnel, something went wrong, exiting..."
    exit 1
fi

echo ""
echo "Created tunnel on ${TCP_SSH_TUNNEL_ADDRESS}..."

# Sleep a bit so the cluster can start correctly.
echo "Sleeping 5s to give the ${TCP_SSH_TUNNEL_ADDRESS} SSH tunnel time to connect..."
sleep 5

# Register CRDs.
echo "Registering CRDs..."
kubectl apply -f ${CURRENT_DIR}/../../test/integration/crd/manifests

# Run tests.
echo "Run tests..."
export TEST_WEBHOOK_URL="https://${TCP_SSH_TUNNEL_ADDRESS}"
export TEST_LISTEN_PORT=${LOCAL_PORT}
    ${CURRENT_DIR}/integration-test.sh
