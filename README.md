[![GoDoc](https://godoc.org/github.com/theupdateframework/notary?status.svg)](https://godoc.org/github.com/theupdateframework/notary)
[![Circle CI](https://circleci.com/gh/theupdateframework/notary/tree/master.svg?style=shield)](https://circleci.com/gh/theupdateframework/notary/tree/master) [![CodeCov](https://codecov.io/github/theupdateframework/notary/coverage.svg?branch=master)](https://codecov.io/github/theupdateframework/notary) [![GoReportCard](https://goreportcard.com/badge/theupdateframework/notary)](https://goreportcard.com/report/github.com/theupdateframework/notary)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftheupdateframework%2Fnotary.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftheupdateframework%2Fnotary?ref=badge_shield)

# Notice

This repository provides an implementation of 
*[The Update Framework specification](https://github.com/theupdateframework/specification)* 
and all references to `notary` in this repository refer to the implementation of the client 
and server aligning with the [TUF](https://github.com/theupdateframework/specification) specification.
The most prominent use of this implementation is in Docker Content Trust (DCT).
The first release [v0.1](https://github.com/notaryproject/notary/releases/tag/v0.1) was released in November, 2015.

# Overview

This repository comprises of a [server](cmd/notary-server) and a [client](cmd/notary) for running and interacting
with trusted collections. See the [service architecture](docs/service_architecture.md) documentation
for more information.

The aim is to make the internet more secure by making it easy for people to
publish and verify content. We often rely on TLS to secure our communications
with a web server, which is inherently flawed, as any compromise of the server
enables malicious content to be substituted for the legitimate content.

Publishers can sign their content offline using keys kept highly
secure. Once the publisher is ready to make the content available, they can
push their signed trusted collection to the `notary` server.

Consumers, having acquired the publisher's public key through a secure channel,
can then communicate with any `notary` server or (insecure) mirror, relying
only on the publisher's key to determine the validity and integrity of the
received content.

## Goals

The `notary` client and server is based on [The Update Framework](https://www.theupdateframework.com/), a secure general design for the problem of software distribution and updates. By using TUF, the `notary` client and server achieves a number of key advantages:

* **Survivable Key Compromise**: Content publishers must manage keys in order to sign their content. Signing keys may be compromised or lost so systems must be designed in order to be flexible and recoverable in the case of key compromise. TUF's notion of key roles is utilized to separate responsibilities across a hierarchy of keys such that loss of any particular key (except the root role) by itself is not fatal to the security of the system.
* **Freshness Guarantees**: Replay attacks are a common problem in designing secure systems, where previously valid payloads are replayed to trick another system. The same problem exists in the software update systems, where old signed can be presented as the most recent. Notary makes use of timestamping on publishing so that consumers can know that they are receiving the most up to date content. This is particularly important when dealing with software update where old vulnerable versions could be used to attack users.
* **Configurable Trust Thresholds**: Oftentimes there are a large number of publishers that are allowed to publish a particular piece of content. For example, open source projects where there are a number of core maintainers. Trust thresholds can be used so that content consumers require a configurable number of signatures on a piece of content in order to trust it. Using thresholds increases security so that loss of individual signing keys doesn't allow publishing of malicious content.
* **Signing Delegation**: To allow for flexible publishing of trusted collections, a content publisher can delegate part of their collection to another signer. This delegation is represented as signed metadata so that a consumer of the content can verify both the content and the delegation.
* **Use of Existing Distribution**: Notary's trust guarantees are not tied at all to particular distribution channels from which content is delivered. Therefore, trust can be added to any existing content delivery mechanism.
* **Untrusted Mirrors and Transport**: All of the notary metadata can be mirrored and distributed via arbitrary channels.

## Security

Any security vulnerabilities can be reported to security@docker.com.

See [service architecture docs](docs/service_architecture.md#threat-model) for more information about our threat model, which details the varying survivability and severities for key compromise as well as mitigations.

### Security Audits

Below are the two public security audits:

* [August 7, 2018 by Cure53](docs/resources/cure53_tuf_notary_audit_2018_08_07.pdf) covering TUF and the `notary` client and server.
* [July 31, 2015 by NCC](docs/resources/ncc_docker_notary_audit_2015_07_31.pdf) covering `notary` client and server.

# Getting started with the notary CLI

Get the `notary` client CLI binary from [the official releases page](https://github.com/theupdateframework/notary/releases) or you can [build one yourself](#building-notary).
The version of the `notary` server and signer should be greater than or equal to notary CLI's version to ensure feature compatibility (ex: CLI version 0.2, server/signer version >= 0.2), and all official releases are associated with GitHub tags.

To use the notary CLI with Docker hub images, have a look at notary's
[getting started docs](docs/getting_started.md).

For more advanced usage, see the
[advanced usage docs](docs/advanced_usage.md).

To use the CLI against a local `notary` server rather than against Docker Hub:

1. Ensure that you have [docker and docker-compose](https://docs.docker.com/compose/install/) installed.
1. `git clone https://github.com/theupdateframework/notary.git` and from the cloned repository path,
    start up a local `notary` server and signer and copy the config file and testing certs to your
    local notary config directory:

    ```sh
    $ docker-compose build
    $ docker-compose up -d
    $ mkdir -p ~/.notary && cp cmd/notary/config.json cmd/notary/root-ca.crt ~/.notary
    ```

1. Add `127.0.0.1 notary-server` to your `/etc/hosts`, or if using docker-machine,
    add `$(docker-machine ip) notary-server`).

You can run through the examples in the
[getting started docs](docs/getting_started.md) and
[advanced usage docs](docs/advanced_usage.md), but
without the `-s` (server URL) argument to the `notary` command since the server
URL is specified already in the configuration, file you copied.

You can also leave off the `-d ~/.docker/trust` argument if you do not care
to use `notary` with Docker images.

## Upgrading dependencies

To prevent mistakes in vendoring the go modules a buildscript has been added to properly vendor the modules using the correct version of Go to mitigate differences in CI and development environment.

Following procedure should be executed to upgrade a dependency. Preferably keep dependency upgrades in a separate commit from your code changes.

```bash
go get -u github.com/spf13/viper
buildscripts/circle-validate-vendor.sh
git add .
git commit -m "Upgraded github.com/spf13/viper"
```

The `buildscripts/circle-validate-vendor.sh` runs `go mod tidy` and `go mod vendor` using the given version of Go to prevent differences if you are for example running on a different version of Go.

## Building notary

Note that the [latest stable release](https://github.com/theupdateframework/notary/releases) is at the head of the
[releases branch](https://github.com/theupdateframework/notary/tree/releases).  The master branch is the development
branch and contains features for the next release.

Prerequisites:

* Go >= 1.12

Set [```GOPATH```](https://golang.org/doc/code.html#GOPATH). Then, run:

```bash
$ export GO111MODULE=on
$ go get github.com/theupdateframework/notary
# build with pkcs11 support by default to support yubikey
$ go install -tags pkcs11 github.com/theupdateframework/notary/cmd/notary
$ notary
```

To build the server and signer, run `docker-compose build`.

## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftheupdateframework%2Fnotary.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftheupdateframework%2Fnotary?ref=badge_large)
