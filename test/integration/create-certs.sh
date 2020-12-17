#! /bin/bash


DOMAINS="*.tcp.ngrok.io tcp.ngrok.io"

OUTPATH=./test/integration/certs
OUTCERT="${OUTPATH}/cert.pem"
OUTKEY="${OUTPATH}/key.pem"

# Create certs for our webhook
mkdir -p "${OUTPATH}"
set -f
mkcert \
    -cert-file "${OUTCERT}" \
    -key-file "${OUTKEY}" \
    ${DOMAINS}
set +f
