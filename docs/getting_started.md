<!--[metadata]>
+++
title = "Getting Started with Notary"
description = "Performing basic operation to use notary in tandem with Docker Content Trust."
keywords = ["docker, notary, notary-client, docker content trust, content trust"]
[menu.main]
parent="mn_notary"
weight=1
+++
<![end-metadata]-->

# Content

- [Who should read this?](#who_should_read_this)
- [Installing Notary](#installing_notary)
- [A Note on Naming](#a_note_on_naming)
- [Inspecting a Docker Hub Repository](#inspecting_a_docker_hub_repository)
- [Deleting a Tag](#deleting_a_tag)
- [Configuring the Client](#configuring_the_client)

# <a name="who_should_read_this">Who should read this?</a>

This document describes basic use of notary as a tool supporting Docker Content Trust. For more advanced use cases you will need to [run your own service](running_a_service.md) and should read the [Advanced Usage](advanced_usage.md) documentation.

# <a name="installing_notary">Installing Notary</a>

A precompiled notary binary for 64 bit Linux or Mac OS X can be downloaded from the repository's [releases page on GitHub](https://github.com/docker/notary/releases). Windows is not officially supported at the moment but if you are a developer and Windows user, we would appreciate any insight you can provide regarding issues.

# <a name="a_note_on_naming">A Note on Naming</a>

Notary uses Globally Unique Names (GUNs) to identify trust repositories. To enable notary to run in a multi-tenant fashion, it requires that all docker image names are provided in their Globally Unique Name format, which is defined as:

- For official images (identifiable by the "Official Repository" moniker), the image name as displayed on Docker Hub, prefixed with `docker.io/library/`. i.e. if you would normally type `docker pull ubuntu` you must enter `notary <cmd> docker.io/library/ubuntu`.
- For all other images, the image name as displayed on hub, prefixed by `docker.io`.

The docker client takes care of these name expansions for you under the hood so do not change the names you use with docker. This is a requirement only when interacting with the same Docker Hub repositories through the notary client.

# <a name="inspecting_a_docker_hub_repository">Inspecting a Docker Hub Repository</a>

The most basic operation is listing the available signed tags in a repository. The notary client used in isolation does not know where the trust repositories are located so we must provide the `-s` (or long form `--server`) flag to tell the client what server it should communicate with. The official Docker Hub notary servers are located at `https://notary.docker.io`. Additionally, notary stores your own signing keys, and a cache of previously downloaded trust metadata in a directory, provided with the `-d` flag. When interacting with Docker Hub repositories, you must instruct the client to use the associated trust directory, which by default is found at `.docker/trust` within the calling user's home directory (failing to use this directory may result in errors when publishing updates to your trust data):

```
$ notary -s https://notary.docker.io -d ~/.docker/trust list docker.io/library/alpine
   NAME                                 DIGEST                                SIZE (BYTES)    ROLE
------------------------------------------------------------------------------------------------------
  2.6      e9cec9aec697d8b9d450edd32860ecd363f2f3174c8338beb5f809422d182c63   1374           targets
  2.7      9f08005dff552038f0ad2f46b8e65ff3d25641747d3912e3ea8da6785046561a   1374           targets
  3.1      e876b57b2444813cd474523b9c74aacacc238230b288a22bccece9caf2862197   1374           targets
  3.2      4a8c62881c6237b4c1434125661cddf09434d37c6ef26bf26bfaef0b8c5e2f05   1374           targets
  3.3      2d4f890b7eddb390285e3afea9be98a078c2acd2fb311da8c9048e3d1e4864d3   1374           targets
  edge     878c1b1d668830f01c2b1622ebf1656e32ce830850775d26a387b2f11f541239   1374           targets
  latest   24a36bbc059b1345b7e8be0df20f1b23caa3602e85d42fff7ecd9d0bd255de56   1377           targets
```

The output shows us the names of the tags available, the hex encoded sha256 digest of the image manifest associated with that tag, the size of the manifest, and the notary role that signed this tag into the repository. The "targets" role is the most common role in a simple repository. When a repository has (or expects) to have collaborators, you may see other "delegated" roles listed as signers, based on the choice of the administrator as to how they organize their collaborators.

> When you run a `docker pull` command, docker is using an integrated notary library, the same one the notary CLI uses, to request the mapping of tag to sha256 digest for the one tag you are interested in (or if you passed the `--all` flag it will use the list operation to efficiently retrieve all the mappings). Having validated the signatures on the trust data, the client will then instruct the engine to do a "pull by digest" in which the docker engine uses the sha256 checksum as a content address to request and validate the image manifest from the docker registry.

# <a name="deleting_a_tag">Deleting a Tag</a>

Notary generates and stores signing keys on the host it's running on. This means that the Docker Hub cannot delete tags from the trust data, they must be deleted using the notary client. This can be done with the `notary remove` command. Again, we must direct it to speak to the correct notary server (N.B. neither you nor the author has permissions to delete tags from the official alpine repository, the output below is for demonstration only):

```
$ notary -s https://notary.docker.io -d ~/.docker/trust remove docker.io/library/alpine 2.6
Removal of 2.6 from docker.io/library/alpine staged for next publish.
```

Note the output message indicates that the removal has only been staged. When performing any write operations they are staged into a changelist, which is applied to the latest version of the trust repository the next time a `notary publish` is run for that repository. You can see the pending change by running `notary status` for the modified repository. The `status` subcommand is an offline operation and as such, does not require the `-s` flag, however it will silently ignore the flag if provided. Failing to provide the correct value for the `-d` flag may show the wrong (probably empty) changelist:

```
$ notary -d ~/.docker/trust status docker.io/library/alpine
Unpublished changes for docker.io/library/alpine:

action    scope     type        path
----------------------------------------------------
delete    targets   target      2.6
$ notary -s https://notary.docker.io publish docker.io/library/alpine
```

# <a name="configuring_the_client">Configuring the Client</a>

It is verbose and tedious to always have to provide the `-s` and `-d` flags manually to most commands. A simple way to create preconfigured versions of the notary command is via aliases. Add the following to your `.bashrc` or equivalent:

```
alias dockernotary="notary -s https://notary.docker.io -d ~/.docker/trust"
```

More advanced methods of configuration, and additional options, can be found in the [configuration doc](configuration.md)
