<!--[metadata]>
+++
title = "Notary Command Reference"
description = "Notary command reference"
keywords = ["docker, notary, command, notary command, reference"]
[menu.main]
parent="mn_notary"
weight=99
+++
<![end-metadata]-->

# Notary Command Reference

## Terminology Reference
1. **GUN**: Notary uses Globally Unique Names (GUNs) to identify trusted collections.
2. **Target**: Notary refers to the files the framework is used to distribute as "target files".  Target files are opaque to the framework. Whether target files are packages containing multiple files, single text files, or executable binaries is irrelevant to Notary.
3. **Trusted Collection**: A trusted collection is a conceptual source of target files of interest to the application.
4. **Key roles**:
    - **Root**: The root role delegates trust to specific keys trusted for all other top-level roles used in the system.
    - **Targets**: The targets role's signature indicates which target files are trusted by clients.
    - **Delegations**: the targets role can delegate full or partial trust to other roles.  Delegating trust means that the targets role indicates another role (that is, another set of keys and the threshold required for trust) is trusted to sign target file metadata.
    - **Snapshot**:  The snapshot role signs a metadata file that provides information about the latest version of all of the other metadata on the trusted collection (excluding the timestamp file, discussed below).
    - **Timestamp**:  To prevent an adversary from replaying an out-of-date signed metadata file whose signature has not yet expired, an automated process periodically signs a timestamped statement containing the hash of the snapshot file.

To read further about the framework Notary implements, check out [The Update Framework](https://github.com/theupdateframework/tuf/blob/develop/docs/tuf-spec.txt)

## Command Reference

### Set up Notary CLI

Once you install the Notary CLI client, you can use it to manage your signing
keys, authorize other team members to sign content, and rotate the keys if
a private key has been compromised.

When using the Notary CLI client you need to specify where is Notary server
you want to communicate with with the `-s` flag, and where to store the private keys and cache for
the CLI client with the `-d` flag.  There are also fields in the [client configuration](/reference/client-config.md) to set these variables.

```bash
# Create an alias to always have the notary client talking to the right notary server
$ alias notary="notary -s <notary_server_url> -d <notary_cache_directory>
```

When working Docker Content Trust, it is important to connect to Docker Hub's Notary server located at `https://notary.docker.io`, and specify notary's metadata cache as `~/.docker/trust`.

## Initializing a trusted collection

Notary can initialize a trusted collection with the `notary init` command:
```bash
$ notary init <GUN>
```

This command will generate targets and snapshot keys locally for the trusted collection, and try to locate a root key to use in the specified metadata cache (from `-d` or the config).
If notary cannot find a root key, it will generate one.  For all keys, notary will also prompt for a passphrase to encrypt the private key material at rest.

If you'd like to initialize your trusted collection with a specific root key, there is a flag:
```bash
$ notary init <GUN> --rootkey <key_file>
```

Note that you will have to run a publish after this command, as there are staged changes to initialize the trusted collection that have not been pushed to a ntoary server:
```bash
$ notary publish <GUN>
```

## Auto-publish changes

Instead of manually running `notary publish` after each command, you can use the `-p` flag to auto-publish the changes from that command.
For example:

```bash
$ notary init -p <GUN>
```

## Manage staged changes

The Notary CLI client stages changes before publishing them to the server.
You can manage the changes that are staged by running:

```bash
# Check what changes are staged
$ notary status <GUN>

# Unstage a specific change
$ notary status <GUN> --unstage 0

# Alternatively, unstage all changes
$ notary status <GUN> --reset
```

Note that `<GUN>` can take on arbitrary values, but when working with Docker Content Trust they are structured like `<url>/<account>/<repository>`.  For example `docker.io/library/alpine` for the [Docker Library Alpine image](https://hub.docker.com/r/library/alpine).

When you're ready to publish your changes to the Notary server, run:

```bash
$ notary publish <GUN>
```


## Delete trust data

Users can remove all notary signed data for a trusted collection by running:

```bash
$ notary delete <GUN> --remote
```

If you don't include the `--remote` flag, Notary deletes local cached content
but will not delete data from the Notary server.

## Change the passphrase for a key

The Notary CLI client manages the keys used to sign the trusted collection. These keys are encrypted at rest.
To list all the keys managed by the Notary CLI client, run:

```bash
$ notary key list
```

To change the passphrase used to encrypt one of the keys, run:

```bash
$ notary key passwd <key_id>
```

## Rotate keys

If one of the private keys is compromised you can rotate that key, so that
content that was signed with those keys stop being trusted.

For keys that are kept offline and managed by the Notary CLI client, such the
keys with the root, targets, and snapshot roles, you can rotate them with:

```bash
$ notary key rotate <GUN> <key_role>
```

The Notary CLI client generates a new key for the role you specified, and
prompts you for a passphrase to encrypt it.
Then you're prompted for the passphrase for the key you're rotating, and if it
is correct, the Notary CLI client contacts the Notary server to publish the
change.

You can also rotate keys that are stored in the Notary server, such as the keys
with the snapshot or timestamp role. For that, run:

```bash
$ notary key rotate <GUN> <key_role> -r
```

## Importing and exporting keys

Notary can import keys that are already in a PEM format:
```bash
$ notary key import <pemfile> --role <key_role> --gun <key_gun>
```

If no `--role` or `--gun` is given, notary will assume that the key is to be used for a delegation role.
Moreover, it's possible for notary to import multiple keys contained in one PEM file, each separated into separate PEM blocks.

For each key it attempts to import, notary will prompt for a passphrase so that the key can be encrypted at rest.

Notary can also export all of its encrypted keys, or individually by key ID or GUN:
```bash
# export all my notary keys to a file
$ notary key export -o exported_keys.pem

# export a single key by ID
$ notary key export --key <keyID> -o exported_keys.pem

# export multiple keys by ID
$ notary key export --key <keyID1> --key <keyID2> -o exported_keys.pem

# export all keys for a GUN
$ notary key export --gun <GUN> -o exported_keys.pem

# export all keys for multiple GUNs
$ notary key export --gun <GUN1> --gun <GUN2> -o exported_keys.pem
```
When exporting multiple keys, all keys are outputted to a single PEM file in individual blocks. If the output flag `-o` is omitted, the PEM blocks are outputted to STDOUT.

## Manage keys for delegation roles

To delegate content signing to other users without sharing the targets key, retrieve a x509 certificate for that user and run:

```bash
$ notary delegation add -p <GUN> targets/<role> --all-paths user.pem user2.pem
```

The user can then import the private key for that certificate keypair (using `notary key import`) and use it for signing.

It's possible to add multiple certificates at once for a role:
```bash
$ notary delegation add -p <GUN> targets/<role> --all-paths user1.pem user2.pem user3.pem
```

You can also remove keys from a delegation role:

```bash
# Remove the given keys from a delegation role
$ notary delegation remove -p <GUN> targets/<role> <keyID1> <keyID2>

# Alternatively, you can remove keys from all delegation roles
$ notary delegation purge <GUN> --key <keyID1> --key <keyID2>
```

## Witnessing delegations

Notary can mark a delegation role for re-signing without adding any additional content:

```bash
$ notary witness -p <GUN> targets/<role>
```

This is desirable if you would like to sign a delegation role's existing contents with a new key.

It's possible that this could be useful for auditing, but moreover it can help recover a delegation role that may have become invalid.
For example: Alice last updated delegation `targets/qa`, but Alice since left the company and an administrator has removed her delegation key from the repo.
Now delegation `targets/qa` has no valid signatures, but another signer in that delegation role can run `notary witness targets/qa` to sign off on the existing contents, provided it is still trusted content.

## Troubleshooting

Notary CLI has a `-D` flag that you can use to increase the logging level. You
can use this for troubleshooting.

Usually most problems are fixed by ensuring you're communicating with the
correct Notary server, using the `-s` flag, and that you're using the correct
directory where your private keys are stored, with the `-d` flag.

If you are receiving this error:
```bash
* fatal: Get <URL>/v2/: x509: certificate signed by unknown authority
```
The Notary CLI must be configured to trust the root certificate authority of the server it is communicating with.
This can be configured by specifying the root certificate with the `--tlscacert` flag or by editing the Notary client configuration file.