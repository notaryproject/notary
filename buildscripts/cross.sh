#!/usr/bin/env bash

# This script cross-compiles static (when possible) binaries for supported OS's
# architectures.  The Linux binary is completely static, whereas Mac OS binary
# has libtool statically linked in. but is otherwise not static because you
# cannot statically link to system libraries in Mac OS.

GOARCH="amd64"

if [[ "${NOTARY_BUILDTAGS}" == *pkcs11* ]]; then
	export CGO_ENABLED=1
else
	export CGO_ENABLED=0
fi


for os in "$@"; do
	export GOOS="${os}"

	if [[ "${GOOS}" == "darwin" ]]; then
		export CC="o64-clang"
		export CXX="o64-clang++"
		# -ldflags=-s:  see https://github.com/golang/go/issues/11994 - TODO: this has been fixed in go 1.7.1
		# darwin binaries can't be compiled to be completely static with the -static flag
		LDFLAGS="-s"
	else
		unset CC
		unset CXX
		LDFLAGS="-extldflags -static"
	fi

	mkdir -p "${NOTARYDIR}/cross/${GOOS}/${GOARCH}";

	set -x;
	go build \
		-o "${NOTARYDIR}/cross/${GOOS}/${GOARCH}/notary" \
		-a \
		-tags "${NOTARY_BUILDTAGS}" \
		-ldflags "-w ${CTIMEVAR} ${LDFLAGS}"  \
		./cmd/notary;
	set +x;
done
