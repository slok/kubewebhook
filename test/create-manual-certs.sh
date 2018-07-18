#! /bin/bash

CN=${1:-"toilet-admission-webhooks.toilet.svc"}
OUTPATH=${2-"./certs"}
OUT_KEY=${OUTPATH}/key.pem
OUT_CERT=${OUTPATH}/cert.pem

# Create certs for our webhook
mkdir -p ${OUTPATH}
openssl genrsa -out ${OUT_KEY} 2048
openssl req -new -key ${OUT_KEY} -subj "/CN=${CN}" -out ./webhookCA.csr 
openssl x509 -req -days 365 -in webhookCA.csr -signkey ${OUT_KEY} -out ${OUT_CERT}
rm ./webhookCA.csr 


echo "* certificate path: ${OUT_CERT}"
echo "* key path: ${OUT_KEY}"
echo "webhook k8s manifest CABundle:"
echo  $(cat ${OUT_CERT} | base64 -w0)
