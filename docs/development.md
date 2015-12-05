<!--[metadata]>
+++
title = "Notary Development"
description = "Development details about Notary"
keywords = ["docker, notary, trust, image, signing, repository, cli"]
[menu.main]
parent="mn_notary"
+++
<![end-metadata]-->

# Building Docker Images

The notary repository comes with Dockerfiles and a docker-compose file
to facilitate development.

## Using `docker-compose`
The compose file specifies a Notary Server that depends upon a MySQL backend
and a remote signing service, which is specified by a Notary Signer that
depends upon the same MySQL backend.

Simply run the following commands to start the server, the signer, and a
temporary MySQL database in containers:

```
$ docker-compose build
$ docker-compose up
```

If you are on Mac OSX with docker-machine or kitematic, you'll need to
update your hosts file such that the name `notary-server` is associated with
the IP address of your VM (for docker-machine, this can be determined
by running `docker-machine ip default`; with kitematic, `echo $DOCKER_HOST`
should show the IP of the VM). If you are using the default Linux setup,
you need to add `127.0.0.1 notary-server` to your hosts file.

## Standalone server

To run a standalone server, without running a signer or mysql, you can use
the following command (after running `docker-compose build` from above):

```
$ docker run --rm -p 4443:4443 -e NOTARY_SERVER_TRUST_SERVICE_TYPE=local \
    notary_notaryserver
```

## Successfully connecting to the container over TLS

By default notary-server runs with TLS with certificates signed by a local
CA. In order to be able to successfully connect to it using
either `curl` or `openssl`, you will have to use the root CA file in
`fixtures/root-ca.crt`.

OpenSSL example:

`openssl s_client -connect notary-server:4443 -CAfile fixtures/root-ca.crt`


# Compiling binaries

Prerequisites:

- Go = 1.5.1
- [godep](https://github.com/tools/godep) installed
- libtool development headers installed

    `notary-signer` depends upon `pkcs11`, which requires the libtool headers.

    Install `libtool-dev` on Ubuntu, `libtool-ltdl-devel` on CentOS/RedHat.
    If you are using Mac OS, you can `brew install libtool`.

Install the go dependencies by running `godep restore`.

From the root of the notary git repository, run `make binaries`. This will
compile the `notary`, `notary-server`, and `notary-signer` applications and
place them in a `bin` directory at the root of the git repository (the `bin`
directory is ignored by the .gitignore file).

Assuming a standard installation of Homebrew, you may find that you need the
following environment variables:

```sh
export CPATH=/usr/local/include:${CPATH}
export LIBRARY_PATH=/usr/local/lib:${LIBRARY_PATH}
```

## Running the binaries

Both the server and signer have the following usage

```
$ bin/notary-<server|signer> --help
usage: bin/notary-<server|signer>
  -config="": Path to configuration file
  -debug=false: Enable the debugging server on localhost:8080
```

Please see [the Notary Server config docs](notary-server-config.md) and
[the Notary Signer config docs](notary-signer-config.md) to learn about the
format of the configuration file.

The pem and key provided in fixtures are purely for local development and
testing. For production, you must create your own keypair and certificate,
either via the CA of your choice, or a self signed certificate.

If using the pem and key provided in fixtures, then when using the notary client
binary (`bin/notary`):

- Add `fixtures/root-ca.crt` to your trusted root certificates
- Use the default configuration for notary client that loads the CA root for
  you by using the flag `-c ./cmd/notary/config.json`
- Disable TLS verification by adding the following option notary configuration
  file in `~/.notary/config.json`:

       "skipTLSVerify": true

Otherwise, you will see TLS errors or X509 errors upon initializing the
notary collection:

```
$ notary list diogomonica.com/openvpn
* fatal: Get https://notary-server:4443/v2/: x509: certificate signed by unknown authority
$ notary list diogomonica.com/openvpn -c cmd/notary/config.json
latest b1df2ad7cbc19f06f08b69b4bcd817649b509f3e5420cdd2245a85144288e26d 4056
```
