<!--[metadata]>
+++
title = "Basic Operations"
description = "Basic operations using Notary"
keywords = ["docker, notary, trust, image, signing, repository, cli"]
[menu.main]
parent="mn_notary"
+++
<![end-metadata]-->

# Basic operations using Notary

The notary CLI provides a convenient way of managing trusted collections and the keys associated with them. On this page we describe a high level overview of the functionality of each subcommand. For a quick walk-through of the basic Notary functionality check out [getting started with Notary](gettingstarted.md).

## Creating a new collection of data

Before you create a new collection of data, you have to name it. In notary, each collection of data is identified by a Global Unique Name (GUN). GUNs can be arbitrary, but they have to be unique. An easy way of having unique GUNs is by including any other pre-existing global namespace already available such as DNS. A good example of a GUN is: example.com/user/collection.

After chosing a name, you can initialize a repository by running the `init` command:

```
$ notary init example.com/user/collection
```

On initialization, notary creates all the necessary keys needed to sign content for this collection, create the necessary TUF metadata files and push them to the remote notary server. If you already have a root key, notary uses it instead of generating a new one.

The default notary server URL configured with Notary is [https://notary-server:4443/]. This default value can overridden (by priority order):

	* by specifying the option `--server/-s` on commands requiring call to the notary server.
	* by setting the `NOTARY_SERVER_URL` environment variable.
	* by setting the `remote_server` option inside of your configuration file (defaults to `~/.notary/config.json)

## Adding and deleting content from an existing trusted collection

After you initialize a trusted collection you can add and remove any content from it. You can do this by running the `add` and `remove` commands, respectively:

```
$ notary add example.com/user/collection TARGET FILEPATH
$ notary remove example.com/user/collection TARGET
```

The `add` command takes the Global Unique Name of the trusted collection, a target name, and the path to the file you want to add.
The `remove` command takes the Global Unique Name of the trusted collection and the name of the target you want to remove.

When running `add` notary calculates a cryptographic hash over the file in `FILEPATH` and point the `TARGET` name to it. Neither the `add` or `remove` commands require access to the remote notary server, since all changes are staged locally for later publishing.

## Inspecting staged changes

You can inspect the changes notary has pending locally for a particular trusted collection by running the `status` command:

```
$ notary status example.com/user/collection
```

These changes are kept locally until you publish this collection to the remote notary server.

## Publishing changes

In order for your local changes to be see by other notary clients, you have to publish them to a notary server. This can be done by using the `publish` command:

```
$ notary publish example.com/user/collection
```

When publishing, notary finds the keys necessary to sign a particular trusted collection, uses them to create a signed snapshot and uploads it to the remote notary server.

## Listing targets

You can list the targets that are currently part of a trusted collection by running the `list` command:

```
$ notary list example.com/user/collection
```

The `list` command will attempt to connect to the remote notary server and retrieve the most up-to-date version of the targets for a trusted collection. In case the connection fails, notary will use the locally cached metadata for up to two weeks, period after which it will start returning an error due to an expired cache.


## Resolving a specific target

Instead of listing all the available targets for a particular collection, you also can retrieve a particular target by using the `lookup` command:

```
$ notary lookup example.com/user/collection TARGET
```

## Verifying content

You can verify if a particular file matches a TARGET in a trusted collection by using the `verify` command:

```
$ cat FILEPATH | notary verify example.com/user/collection TARGET
```

The `verify` command receives data via the STDIN, calculates a cryptographic hash of the content, and checks it against the remote trusted collection. If the content matches, `notary verify` will send to STDOUT exactly what it received in STDIN, allowing sequencing of trusted operations on a command line:

```
curl -ssL http://example.com/installer.sh | notary verify example.com/user/collection TARGET | sh
```

## Notary Server

The default notary server URL is [https://notary-server:4443/]. This default value can overridden (by priority order):

- by specifying the option `--server/-s` on commands requiring call to the notary server.
- by setting the `NOTARY_SERVER_URL` environment variable.
