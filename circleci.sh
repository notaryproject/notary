#!/usr/bin/env bash

if [ "${CIRCLE_NODE_TOTAL}" -lt 4 ] {
	echo "Expected CircleCI to run 4 parallel jobs."
	exit(1)
}

case $CIRCLE_NODE_INDEX in
	0) NOTARY_BUILDTAGS=pkcs11 PKGS="github.com/docker/notary/client" make ci
	   ;;
	1) NOTARY_BUILDTAGS=none PKGS="github.com/docker/notary/client" make ci
	   ;;
	2) NOTARY_BUILDTAGS=pkcs11 EXCLUDE_PKGS="github.com/docker/notary/client" make ci
	   ;;
	3) NOTARY_BUILDTAGS=none EXCLUDE_PKGS="github.com/docker/notary/client" make ci
	   ;;
esac
