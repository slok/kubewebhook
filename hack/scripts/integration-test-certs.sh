#! /bin/bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# This script will generate the certificates for integration tests
# using mkcert.
DOMAINS='*.serveo.net *.tcp.ngrok.io localhost 127.0.0.1 ::1'

OUTPATH=${CURRENT_DIR}/../../test/integration/certs
OUTCERT=${OUTPATH}/cert.pem
OUTKEY=${OUTPATH}/key.pem

# Create certs for our webhook
set -f
mkcert \
    -cert-file ${OUTCERT} \
    -key-file ${OUTKEY} \
    ${DOMAINS}
set +f