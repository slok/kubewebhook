#!/bin/bash
# vim: ai:ts=8:sw=8:noet
set -eufCo pipefail
export SHELLOPTS
IFS=$'\t\n'

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CRD_MANIFESTS_PATH="${CURRENT_DIR}/../../test/integration/crd/manifests"
TEST_WEBHOOK_URL="${TEST_WEBHOOK_URL:-}"

echo "[*] Make sure you have a running Kubernetes cluster (e.g 'kind create cluster')."
echo "[*] The script will use default KUBECONFIG env var and fallback to default kube config path."

# Check if we are already using a tunnel address

if [ -z "${TEST_WEBHOOK_URL}" ]; then
    echo "[*] Create a TCP tunnel, for example using ngrok like 'ssh -R 0:localhost:8080 tunnel.us.ngrok.com tcp 22'"
    echo "Please enter the tunnel address (e.g '0.tcp.ngrok.io:18776'):"
    read TEST_WEBHOOK_URL
    echo "[*] Remember you can use 'TEST_WEBHOOK_URL' env var to skip this step."
else
    echo "[*] Using '${TEST_WEBHOOK_URL}' from 'TEST_WEBHOOK_URL' env var"
fi

# Sanitize and export correct env var for tests.
TEST_WEBHOOK_URL="$(echo -n ${TEST_WEBHOOK_URL} | sed s/tcp:\\/\\/// )"
TEST_WEBHOOK_URL="$(echo -n ${TEST_WEBHOOK_URL} | sed s/http:\\/\\/// )"
TEST_WEBHOOK_URL="$(echo -n ${TEST_WEBHOOK_URL} | sed s/https:\\/\\/// )"
TEST_WEBHOOK_URL="$(echo -n ${TEST_WEBHOOK_URL} | xargs)" # Trim spaces.
export TEST_WEBHOOK_URL="https://${TEST_WEBHOOK_URL}"

echo "[*] Ensuring integration test CRDs are present on the cluster..."
kubectl apply -f "${CRD_MANIFESTS_PATH}"

echo "[*] Running tests pointing to '${TEST_WEBHOOK_URL}' webhooks..."
"${CURRENT_DIR}"/integration-test.sh