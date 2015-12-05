<!--[metadata]>
+++
draft=true
title = "Overview of Docker Notary"
description = "Overview of Docker Notary"
keywords = ["docker, notary, trust, image, signing, repository"]
[menu.main]
parent="mn_notary"
weight=-99
+++
<![end-metadata]-->

# Overview of Docker Notary

Notary is a [tool for publishing and managing trusted collections of content](basic-operations.md). Publishers can digitally sign collections and consumers can verify integrity and origin of content. This ability is built on a straightforward key management and signing interface to create signed collections and configure trusted publishers.

With notary anyone can provide trust over arbitrary collections of data. Using [The Update Framework (TUF)](http://theupdateframework.com/) as the underlying security framework, it takes care of the operations necessary to create, manage and distribute the metadata necessary to ensure the integrity and freshness of your content.

The notary client publishes and consumes these collections to and from a Notary service, which consists of:
	- a [Server](notary-server.md), which serves as way to aggregate and distibute many trusted collections
	- a [Signer](notary-signer.md) service, which is optionally used by the Server to sign parts of some trusted collections.

## Brief overview of TUF

For a more detailed description of TUF, [please read the full TUF spec](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt).

TUF enumerates the the trust data of a repository in 4 main metadata files, [each of which corresponds to a role with distinct trust responsibilities](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt#L214).  Each role is associated with one or more [public/private key pairs](https://en.wikipedia.org/wiki/Public-key_cryptography#Understanding), which are used to sign that role's metadata file:

- **root** -
	[This role's metadata file](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt#L489) lists the IDs of the root, targets, snapshot, and timestamp public keys.  Clients use these public keys to verify the signatures on all the metadata files in the repository.  A new root metadata file needs to be generated if any of these keys are rotated.

- **targets** -
	[This role's metadata](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt#L678) file lists all the files that are tracked by this repo, their sizes, and their respective [hashes](https://en.wikipedia.org/wiki/Cryptographic_hash_function).  This file is used to verify the integrity of the actual contents of the repository.

	It can also enumerate [delegations](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt#L387) keys and metadata, which are not yet supported by Notary.

- **snapshot** -
	[This role's metadata file](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt#L604) enumerates the filenames, sizes, and hashes of the root, targets, and delegation files in the repository, so a client can verify the integrity of the metadata files they are downloading.

- **timestamp** -
	[This role's metadata file](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt#L827) gives the filename, hash, size, and hash of the latest snapshot metadata file so the client can verify the integrity of the snapshot metadata as well and be sure that the snapshot metadata they downloaded is not stale.


A client who wants to guarantee that they are getting the latest and valid content:

- downloads the timestamp metadata for that content
- downloads the snapshot metadata, using the hash in the timestamp to verify the contents of the snapshot
- downloads the root and target metadata, using the hashes in the snapshot to verify the root and metadata contents
- verifies the signatures on all the metadata files against the keys listed in the root metadata
- verifies the content of a file (that they have downloaded form elsewhere, or already have on disk) against the content hash list supplied by the targets metadata.

The first time a client downloads a notary repository, they pin the root key so that they can verify that the root doesn't change on them in the future.  If the root key is rotated, the root metadata has to be signed by both the old and the new key.

They do not need to pin the targets or snapshot or timestamp keys - these 3 keys can be rotated out transparently so long as the content publisher still has their valid root key.

Please see [this diagram](tuf_update_flow.jpg) for a more complete overview of the TUF metadata update workflow.

## TUF Keys in Notary

As mentioned above, All the TUF public keys are available in the root metadata file.  Private key management is handled by the user and Notary Server:

**Root Key Pair**: The user is responsible for managing the private part of this key pair.  The private key is referred to in various documents and articles as the "offline key" because it's the most important for the user to protect.  This key is the trickiest to rotate, since consumers will save/pin the public part of this key, which is sometimes referred to as "the content publisher's public key".

**Targets Key Pair**: The user is responsible for managing the private part of this key pair.

**Snapshot Key Pair**: Notary supports having either the user manage the snapshot private key, or having the Notary Server manage the private snapshot key. If the user wants to manage the private key, the user is responsible for generating and signingn snapshots.  If the user does not want to manage this key, one will be created and stored in Notary Signer, and Notary Server will generate and then call Notary Signer to sign snapshots.

**Timestamp Key Pair**: Notary Server manages this private key entirely.  It is stored in Notary Signer, and Notary Server is wholly responsible for generating timestamps and calling Notary Signer to sign the timestamps.

You may see references in various documents and articles (such as [this one](https://blog.docker.com/2015/08/content-trust-docker-1-8/)) to a "tagging key", which is just the combination of the private targets key and private snapshots key.  It's referred to as the tagging key for simplicity because they are both needed on the client side to tag data.
