#! /bin/bash


DOMAINS="*.tcp.ngrok.io tcp.ngrok.io"

OUTPATH=./test/manual/certs
OUTCERT="${OUTPATH}/cert.pem"
OUTKEY="${OUTPATH}/key.pem"
OUTCABUNDLE64="${OUTPATH}/ca-bundle.b64"

# Create certs for our webhook
mkdir -p "${OUTPATH}"
set -f
mkcert \
    -cert-file "${OUTCERT}" \
    -key-file "${OUTKEY}" \
    ${DOMAINS}
set +f

echo "Get webhook config CABundle from "${OUTCABUNDLE64}""
echo -n $(cat ${OUTCERT} | base64 -w0) > "${OUTCABUNDLE64}"