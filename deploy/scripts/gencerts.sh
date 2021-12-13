workdir=${1}
keydir=$workdir/keys
mkdir -p $keydir

echo Generating the CA cert and private key to ${keydir}
openssl req -nodes -new -x509 -keyout ${keydir}/ca.key -out ${keydir}/ca.crt -subj "/CN=crane"

echo Generating the private key for the webhook server
openssl genrsa -out ${keydir}/tls.key 2048

# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
echo Signing the CSR, and generating cert into ${keydir}
openssl req -new -key ${keydir}/tls.key -subj "/CN=webhook-service.crane-system.svc" -config ${workdir}/scripts/webhook.csr \
    | openssl x509 -req -CA ${keydir}/ca.crt -CAkey ${keydir}/ca.key -CAcreateserial -out ${keydir}/tls.crt -extensions v3_req -extfile ${workdir}/scripts/webhook.csr