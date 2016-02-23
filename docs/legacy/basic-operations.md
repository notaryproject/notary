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

On initialization, notary creates all the keys necessary to sign content for this collection, creates the TUF metadata files and stages them, ready for a `publish` command to push them to a remote server. If any root keys are present either in an attached Yubikey, or the appropriate directory (most easily added via the `notary key import` command), the `init` command will choose one of the existing root keys to use for this new collection, preferring a key present in a Yubikey, over a key present on disk. 

The default notary server URL configured with Notary is [https://notary-server:4443/]. This default value can overridden (in priority order) by:

	* specifying the option `--server/-s` on commands requiring call to the notary server.
	* setting the `NOTARY_SERVER_URL` environment variable.
	* setting the `remote_server` option inside of your configuration file (notary will attempt to read a config file at the default location `~/.notary/config.json`)

## Adding and deleting content from an existing trusted collection

After you initialize a trusted collection you can add and remove any content from it. You can do this by running the `add` and `remove` commands, respectively:

```
$ notary add example.com/user/collection TARGET FILEPATH
$ notary remove example.com/user/collection TARGET
```

Both commands operate only on local data, requiring no connection to a notary server, and simply staging changes in a time ordered list to be applied the next time a `publish` is run.
The `add` command takes the Global Unique Name of the trusted collection, a target name, and the path to the file you want to add. It only accepts a single filepath per invocation as only one file may be associated with the `TARGET` name.  Notary calculates a cryptographic hash over the file in `FILEPATH`, and along with the size (in bytes) of the file, associates the `TARGET` name with this metadata. 

The `remove` command takes the Global Unique Name of the trusted collection and the name of the target you want to remove.

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

The `list` command will attempt to connect to the remote notary server and retrieve the most up-to-date version of the targets for a trusted collection. In the case that the connection fails, notary will locally cached metadata up until it expires. The default configuration of notary permits the expiration time to be up to two weeks, although it may be less depending on exactly when the metadata was last signed and retrieved. The `list` command will not include any locally pending changes, instead showing only the state of the remote repository, as visible to the rest of the world.


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
