<!--[metadata]>
+++
title = "Understand the typical commands"
description = "Description of key notary commands"
keywords = ["docker, notary, notary-client"]
+++
<![end-metadata]-->

Notary is a tool for publishing and managing trusted collections of content. Publishers can digitally sign collections and consumers can verify integrity and origin of content. This ability is built on a straightforward key management and signing interface to create signed collections and configure trusted publishers.

With Notary anyone can provide trust over arbitrary collections of data. Using The Update Framework (TUF) as the underlying security framework, Notary takes care of the operations necessary to create, manage and distribute the metadata necessary to ensure the integrity and freshness of your content.

# Notary terminology 

1. Collection: A collection is a conceptual source of target files of interest to the application.
2. GUN: Notary uses Globally Unique Names (GUNs) to identify trust collections
3. Target: we will refer to the files the framework is used to distribute as "target files".  Target files are opaque to the framework. Whether target files are packages containing multiple files, single text files, or executable binaries is irrelevant to Notary 
4. Key roles: 
    - Root = The root role delegates trust to specific keys trusted for all other top-level roles used in the system.
    - Targets = The targets role's signature indicates which target files are trusted by clients. 
        - Delegation roles = the targets role can delegate full or partial trust to other roles.  Delegating trust means that the targets role indicates another role (that is, another set of keys and the threshold required for trust) is trusted to sign target file metadata.
    - Snapshot =  The snapshot role signs a metadata file that provides information about the latest version of all of the other metadata on the repository (excluding the timestamp file, discussed below).
    - Timestamp =  To prevent an adversary from replaying an out-of-date signed metadata file whose signature has not yet expired, an automated process periodically signs a timestamped statement containing the hash of the snapshot file.

To read further about the framework Notary is inspired by, check out [The Update Framework](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt)

# Understand the typical notary commands

## Initialize a collection
`notary init examplegun/image`

*Notary consists of a server, signer and a client. When changes are made on the client to a collection, they are stashed as a changelist locally. To push these changes to the server, a user should publish that collection!*

## View status (unpublished changes) of a collection
`notary status examplegun/image`

## Publish a changelist
`notary publish examplegun/image`

You could also include a `-p` flag to notary commands to auto-publish commands that stage changes as changelists 

example: `notary init -p examplegun/image`

## Add delegations
`notary delegation add examplegun/image targets/qa_team --all-paths <key>`

## Remove delegations
`notary delegation remove examplegun/image targets/qa_team`

## Export private key
`notary key export --key <keyID> -o my_key.pem`

## Import private key
`notary key import my_key.pem --role targets/qa_team`

## Add delegation public key
`notary delegation add examplegun/image targets/qa_team --all-paths <key>`

## Remove delegation public key
`notary delegation remove examplegun/image targets/qa_team`

## Add a target 
`docker push examplegun/image`
or using only notary CLI

`notary add examplegun/image v1:0 file.txt`

## Remove a target
`notary remove examplegun/image v1:0`

## Verify that a given piece of data is a part of a notary collection
`notary verify examplegun -i file.txt v1`

## Rotate a key (root, target, snapshot, timestamp)
`notary key rotate examplegun/image targets`

## Rotate a delegation key
### Admin actions
`notary delegation add examplegun/image targets/qa_team <newkey>`

`notary delegation remove examplegun/image targets/qa_team <oldkey>`

### Owner of old key after admin actions (Not always necessary depending on state)
`notary witness examplegun/image targets/qa_team`

Witness marks roles to be re-signed the next time they're published. It recovers a role that has been made invalid by all keys being removed. It's only necessary if, for example, key4 was the current signer of targets/qa_team and its removal made the role's targets file invalid

## Add a hash as a target to a collection
`notary addhash examplegun/image v1 <byte size> <hashes>`

## Recover from key revocations by marking roles to be re-signed once published
`notary witness examplegun/image targets/qa_team`

## Lookup the hash for a particular target 
`notary lookup examplegun/image v1`

## Delete the trust data for a collection
`notary delete examplegun/image`

Note: To delete trust data from the notary server, you should add the `--remote` flag