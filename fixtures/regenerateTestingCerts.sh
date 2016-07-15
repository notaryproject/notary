#!/usr/bin/env bash

# Script to be used for generating testing certs only for notary-server and notary-signer
# Will also create a root-ca and intermediate-ca, deleting those keys when finished

OPENSSLCNF=
for path in /etc/openssl/openssl.cnf /etc/ssl/openssl.cnf /usr/local/etc/openssl/openssl.cnf; do
    if [[ -e ${path} ]]; then
        OPENSSLCNF=${path}
    fi
done
if [[ -z ${OPENSSLCNF} ]]; then
    printf "Could not find openssl.cnf"
    exit 1
fi

# First generates root-ca
openssl genrsa -out "root-ca.key" 4096
openssl req -new -key "root-ca.key" -out "root-ca.csr" \
        -subj '/C=US/ST=CA/L=San Francisco/O=Docker/CN=Notary Testing CA'

cat > "root-ca.cnf" <<EOL
[root_ca]
basicConstraints = critical,CA:TRUE,pathlen:1
keyUsage = critical, nonRepudiation, cRLSign, keyCertSign
subjectKeyIdentifier=hash
EOL

openssl x509 -req -days 3650 -in "root-ca.csr" -signkey "root-ca.key" \
        -out "root-ca.crt" -extfile "root-ca.cnf" -extensions root_ca
cp "root-ca.crt" "../cmd/notary/root-ca.crt"

rm "root-ca.cnf" "root-ca.csr"

# Then generate intermediate-ca
openssl genrsa -out "intermediate-ca.key" 4096
openssl req -new -key "intermediate-ca.key" -out "intermediate-ca.csr" \
        -subj '/C=US/ST=CA/L=San Francisco/O=Docker/CN=Notary Testing CA'

cat > "intermediate-ca.cnf" <<EOL
[intermediate_ca]
authorityKeyIdentifier=keyid,issuer
basicConstraints = critical,CA:TRUE,pathlen:0
extendedKeyUsage=serverAuth,clientAuth
keyUsage = critical, nonRepudiation, cRLSign, keyCertSign
subjectKeyIdentifier=hash
EOL

openssl x509 -req -days 3650 -in "intermediate-ca.csr" -CA "root-ca.crt" -CAkey "root-ca.key"  -CAcreateserial \
        -out "intermediate-ca.crt" -extfile "intermediate-ca.cnf" -extensions intermediate_ca

rm "intermediate-ca.cnf" "intermediate-ca.csr"
rm "root-ca.key" "root-ca.srl"

# Then generate notary-server
# Use the existing notary-server key
openssl req -new -key "notary-server.key" -out "notary-server.csr" \
        -subj '/C=US/ST=CA/L=San Francisco/O=Docker/CN=notary-server'

cat > "notary-server.cnf" <<EOL
[notary_server]
authorityKeyIdentifier=keyid,issuer
basicConstraints = critical,CA:FALSE
extendedKeyUsage=serverAuth,clientAuth
keyUsage = critical, digitalSignature, keyEncipherment
subjectAltName = DNS:notary-server, DNS:notaryserver, DNS:localhost, IP:127.0.0.1
subjectKeyIdentifier=hash
EOL

openssl x509 -req -days 365 -in "notary-server.csr" -CA "intermediate-ca.crt" -CAkey "intermediate-ca.key"  -CAcreateserial \
        -out "notary-server.crt" -extfile "notary-server.cnf" -extensions notary_server
# append the intermediate cert to this one to make it a proper bundle
cat "intermediate-ca.crt" >> "notary-server.crt"

rm "notary-server.cnf" "notary-server.csr"

# Then generate notary-signer
# Use the existing notary-signer key
openssl req -new -key "notary-signer.key" -out "notary-signer.csr" \
        -subj '/C=US/ST=CA/L=San Francisco/O=Docker/CN=notary-signer'

cat > "notary-signer.cnf" <<EOL
[notary_signer]
authorityKeyIdentifier=keyid,issuer
basicConstraints = critical,CA:FALSE
extendedKeyUsage=serverAuth,clientAuth
keyUsage = critical, digitalSignature, keyEncipherment
subjectAltName = DNS:notary-signer, DNS:notarysigner, DNS:localhost, IP:127.0.0.1
subjectKeyIdentifier=hash
EOL

openssl x509 -req -days 365 -in "notary-signer.csr" -CA "intermediate-ca.crt" -CAkey "intermediate-ca.key"  -CAcreateserial \
        -out "notary-signer.crt" -extfile "notary-signer.cnf" -extensions notary_signer
# append the intermediate cert to this one to make it a proper bundle
cat "intermediate-ca.crt" >> "notary-signer.crt"

rm "notary-signer.cnf" "notary-signer.csr"

# Then generate secure.example.com
# Use the existing secure.example.com key
openssl req -new -key "secure.example.com.key" -out "secure.example.com.csr" \
        -subj '/C=US/ST=CA/L=San Francisco/O=Docker/CN=secure.example.com'

cat > "secure.example.com.cnf" <<EOL
[secure.example.com]
authorityKeyIdentifier=keyid,issuer
basicConstraints = critical,CA:FALSE
extendedKeyUsage=serverAuth,clientAuth
keyUsage = critical, digitalSignature, keyEncipherment
subjectAltName = DNS:secure.example.com, DNS:localhost, IP:127.0.0.1
subjectKeyIdentifier=hash
EOL

openssl x509 -req -days 365 -in "secure.example.com.csr" -CA "intermediate-ca.crt" -CAkey "intermediate-ca.key"  -CAcreateserial \
        -out "secure.example.com.crt" -extfile "secure.example.com.cnf" -extensions secure.example.com
rm "secure.example.com.cnf" "secure.example.com.csr"
rm "intermediate-ca.key" "intermediate-ca.srl"