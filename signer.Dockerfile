FROM golang:1.10.3-stretch

# libltdl-dev needed for PKCS#11 support.
RUN apt-get update
RUN apt-get -y --no-install-recommends install git build-essential libltdl-dev libltdl7

# Some scripts depend on sh=bash (which is a bug but who has the time...)
RUN ln -sf /bin/bash /bin/sh

# Pin to the specific v3.0.0 version
RUN go get -tags 'mysql postgres file' github.com/mattes/migrate/cli && mv /go/bin/cli /go/bin/migrate

ENV NOTARYPKG github.com/theupdateframework/notary

# Copy the local repo to the expected go path
COPY . /go/src/${NOTARYPKG}

WORKDIR /go/src/${NOTARYPKG}

RUN chmod 0600 ./fixtures/database/*

ENV SERVICE_NAME=notary_signer
ENV NOTARY_SIGNER_DEFAULT_ALIAS="timestamp_1"
ENV NOTARY_SIGNER_TIMESTAMP_1="testpassword"

# Install notary-signer
RUN go install \
    -tags pkcs11 \
    -ldflags "-w -X ${NOTARYPKG}/version.GitCommit=`git rev-parse --short HEAD` -X ${NOTARYPKG}/version.NotaryVersion=`cat NOTARY_VERSION`" \
    ${NOTARYPKG}/cmd/notary-signer

# Remove a stack of stuff we don't need
RUN apt-get -y purge git build-essential libltdl-dev gcc g++ python perl subversion openssl openssh-client make
RUN apt-get -y autoremove
RUN rm -rf /var/cache/apt

ENTRYPOINT [ "notary-signer" ]
CMD [ "-config=fixtures/signer-config-local.json" ]
