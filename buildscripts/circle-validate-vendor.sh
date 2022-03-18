#!/usr/bin/env bash

go_version=1.17.8

docker run --rm --env GO111MODULE=on -w /notary --volume ${PWD}:/notary \
    golang:${go_version}-alpine \
    sh -c "apk update && apk add bash git && buildscripts/validate-vendor.sh"
