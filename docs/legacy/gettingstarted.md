<!--[metadata]>
+++
title = "Getting Started"
description = "Getting started with Notary"
keywords = ["docker, notary, trust, image, signing, collection, cli"]
[menu.main]
parent="mn_notary"
+++
<![end-metadata]-->

# Getting Started

On this page you create a trusted collection of data using Notary, add data, and publish it to a remote Notary server.

## Prerequisites

Make sure you have already
[installed Notary](install.md) and are running a [Notary Server](notary-server.md).

## Step 1: Initialization

In order to be able to add content to a trusted collection, you first need to initialize it.

```
$ notary init example.com/collection

No root keys found. Generating a new root key...
You are about to create a new root signing key passphrase. This passphrase
will be used to protect the most sensitive key in your signing system. Please
choose a long, complex passphrase and be careful to keep the password and the
key file itself secure and backed up. It is highly recommended that you use a
password manager to generate the passphrase and keep it safe. There will be no
way to recover this key. You can find the key in your config directory.
Enter passphrase for new root key with ID 1f54328:
Repeat passphrase for new root key with ID 1f54328:
Enter passphrase for new targets key with ID 1df39fc (example.com/collection):
Repeat passphrase for new targets key with ID 1df39fc (example.com/collection):
```

This command initializes a repository named `example.com/collection`, creating a new root key, if one does not exist, and new targets/snapshots keys for this particular collection.

You can now see your keys using the `list` command:

```
$ notary key list

    ROLE              GUN                                          KEY ID                                         LOCATION
--------------------------------------------------------------------------------------------------------------------------------------
  root                                1f5432877ddc386cba5875d7445c4cae7f5ed6476220c84544b850a083bbae7d   file (/tmp/notary1/private)
  snapshot   example.com/collection   e7c927eee376f7f50cc394285b7c9f14146070093836684264c106d1776182d5   file (/tmp/notary1/private)
  targets    example.com/collection   1df39fc0c144a175a3b802370a35279610f818905326bbf3c5f5a92a775a7844   file (/tmp/notary1/private)
```

## Step 2: Add content

In this step, you add content to your trusted collection. When you add a new file to your 
trusted collection, notary will compute a cryptographic hash over its contents and add a
secure mapping between a target (similar to an alias or a tag), and the hash of the content.

```
$ notary add example.com/collection v1 install.sh
Addition of target "v1" to repository "example.com/collection" staged for next publish.
$ notary add example.com/collection latest install.sh
Addition of target "latest" to repository "example.com/collection" staged for next publish.
```

At this point you can use the `status` command to see what files you currently have staged for publishing

```
$ notary status example.com/collection

Unpublished changes for example.com/collection:

action    scope     type        path
----------------------------------------------------
create    targets   target      v1
create    targets   target      latest
```

## Step 3: Publish content

Before any changes to a trusted collection are available to everyone, you have to publish your content, signing all the metadata using the appropriate keys and pushing it to the remote Notary server.

```
$ notary publish example.com/collection
Pushing changes to example.com/collection
Enter passphrase for targets key with ID 1df39fc (example.com/collection):
```

Notary will ask you for the passphrase of the targets/snapshot key pair in order to be able to decrypt the on-disk keys and sign your content.

## Step 4: Re

At this point, you have seen the basics of how Notary works.

## Where to go next

- [Explore Notary's basic operations](basic-operations.md)
- [Notary configuration file reference](notary-config.md)
