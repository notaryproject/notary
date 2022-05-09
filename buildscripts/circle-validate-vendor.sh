#!/usr/bin/env bash

go_version=1.16.9

docker run --rm --env GO111MODULE=on -w /notary --volume ${PWD}:/notary \
    golang:${go_version}-alpine \
    sh -c "apk update && apk add bash git && git config --global safe.directory /notary && buildscripts/validate-vendor.sh"
