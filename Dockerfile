FROM golang:1.4.2-cross

RUN apt-get update && apt-get install -y \
	libltdl-dev \
	libsqlite3-dev \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN go get golang.org/x/tools/cmd/vet \
	&& go get golang.org/x/tools/cmd/cover

COPY . /go/src/github.com/docker/notary

ENV NOTARY_CROSSPLATFORMS  \
	darwin/amd64 \
	freebsd/amd64 \
	windows/amd64

ENV GOPATH /go/src/github.com/docker/notary/vendor:$GOPATH

WORKDIR /go/src/github.com/docker/notary
