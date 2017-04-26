FROM golang:1.7.3-alpine
MAINTAINER David Lawrence "david.lawrence@docker.com"

RUN apk add --update git gcc libc-dev && rm -rf /var/cache/apk/*

# Pin to the specific v1 version
RUN git clone -b v1 https://github.com/mattes/migrate.git /go/src/github.com/mattes/migrate/ && \
    go get github.com/mattes/migrate && \
    go build -tags 'mysql' -o /usr/local/bin/migrate github.com/mattes/migrate

ENV NOTARYPKG github.com/docker/notary

# Copy the local repo to the expected go path
COPY . /go/src/${NOTARYPKG}

WORKDIR /go/src/${NOTARYPKG}

ENV SERVICE_NAME=notary_server
EXPOSE 4443

# Install notary-server
RUN go install \
    -tags pkcs11 \
    -ldflags "-w -X ${NOTARYPKG}/version.GitCommit=`git rev-parse --short HEAD` -X ${NOTARYPKG}/version.NotaryVersion=`cat NOTARY_VERSION`" \
    ${NOTARYPKG}/cmd/notary-server && apk del git gcc libc-dev

ENTRYPOINT [ "notary-server" ]
CMD [ "-config=fixtures/server-config-local.json" ]
