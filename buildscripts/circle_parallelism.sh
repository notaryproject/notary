#!/usr/bin/env bash

set -e

# parallelism disabled here as not working in circleci 2.0

docker run --rm -e NOTARY_BUILDTAGS=pkcs11 --env-file buildscripts/env.list --user notary notary_client bash -c "make ci && codecov"
docker run --rm -e NOTARY_BUILDTAGS=none --env-file buildscripts/env.list --user notary notary_client bash -c "make ci && codecov"
docker run --rm -e NOTARY_BUILDTAGS=pkcs11 notary_client make lint

SKIPENVCHECK=1 make TESTDB=mysql testdb
SKIPENVCHECK=1 make TESTDB=mysql integration

SKIPENVCHECK=1 make TESTDB=rethink testdb
SKIPENVCHECK=1 make TESTDB=rethink integration

SKIPENVCHECK=1 make TESTDB=postgresql testdb
SKIPENVCHECK=1 make TESTDB=postgresql integration

SKIPENVCHECK=1 make cross
