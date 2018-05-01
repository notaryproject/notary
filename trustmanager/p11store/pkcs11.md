# Using Notary with PKCS#11

## Overview

The `pkcs11` key store is always built into Notary.
No build-time configuration is required.

This key store appears (by default) last in the list of supported key stores.
So by default keys will be created in software and no HSM used.
To enable use of an HSM, the new `--keystore` and `--token` arguments must be used
when generating the key.

This key store has been tested with nShield and SoftHSM.

## Device Configuration

You must tell Notary which PKCS#11 provider to use via the environment.

    $ export NOTARY_HSM_LIB=/opt/nfast/toolkits/pkcs11/libcknfast.so

For other devices, specify the path to their PKCS#11 implementation
instead.

## Generating Keys

`notary init` does not generate HSM-protected keys.
Instead you must generate the required keys in advance with `notary key generate`.
This can be done for the root, targets and snapshot keys.

### Root Key

Here I assume a token with a label of `ocs1`.
You can also specify `serialNumber:7e585d361027d0e6`.

    $ notary key generate -K pkcs11 -T label:ocs1 -r root -v
    Enter the passphrase for the PKCS#11 token 'nCipher Corp. Ltd ocs1 (7e585d361027d0e6)':
    Generated new ecdsa root key with keyID: ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca

### Targets Key

    $ notary key generate -K pkcs11 -T label:ocs1 -r targets -g example.com/nshield
    Enter the passphrase for the PKCS#11 token 'nCipher Corp. Ltd ocs1 (7e585d361027d0e6)':
    Generated new ecdsa targets key with keyID: 930e06d6c0f2e0c0e1a45eb403cf573d7d934f44cbd41adec784ead07b7451e5

### Outcome

The keys should be visible in the key list:

    $ notary key list

    ROLE       GUN                    KEY ID                                                              LOCATION
    ----       ---                    ------                                                              --------
    root                              ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca    pkcs11
    targets    example.com/nshield    930e06d6c0f2e0c0e1a45eb403cf573d7d934f44cbd41adec784ead07b7451e5    pkcs11

If you don't set `NOTARY_HSM_LIB` appropriately then the key just disappears:

    $ NOTARY_HSM_LIB="" notary key list -v

    No signing keys found.

## Initializing Collections

`notary init` will automatically pick up a key in the root, targets and snapshot roles if they exist.

    $ notary init example.com/collection
    Root key found, using: ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca
    Enter the passphrase for the PKCS#11 token 'nCipher Corp. Ltd ocs1 (7e585d361027d0e6)':
    Enter passphrase for new targets key with ID 95bf542:
    Repeat passphrase for new targets key with ID 95bf542:
    Enter passphrase for new snapshot key with ID 783c289:
    Repeat passphrase for new snapshot key with ID 783c289:

## Identifying Keys

The `pkcs11` key store encodes the role and GUN into the `CKA_LABEL` attribute,
and the Notary key ID into the `CKA_ID` attribute.
For example:

    CKA_CLASS CKO_PUBLIC_KEY
    CKA_TOKEN true
    CKA_PRIVATE false
    CKA_MODIFIABLE true
    CKA_LABEL "root:"
    CKA_NFKM_APPNAME "pkcs11"
    CKA_NFKM_ID "uc7e585d361027d0e607963663cde62b950dd1a689-18831841f6a00c2501cd2e699df11a33aaddff98"
    CKA_NFKM_HASH length 20
      { F0FC742D 4C915568 1B804FCB 40EB6972 CF751840 }
    CKA_KEY_TYPE CKK_EC
    CKA_ID length 64
      { 65613930 61653734 36636262 33333630 33306462
        33653837 31353231 38636534 64636430 34313165
        36356333 37633830 62663738 39353966 66313737
        34646361 }
      as string "ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca"
    CKA_ISSUER length 0
    CKA_SERIAL_NUMBER length 0
    CKA_DERIVE false
    CKA_LOCAL true
    CKA_START_DATE 0000 00 00
    CKA_END_DATE 0000 00 00
    CKA_KEY_GEN_MECHANISM CKM_EC_KEY_PAIR_GEN
    CKA_ALLOWED_MECHANISMS: ANY
    CKA_SUBJECT length 0
    CKA_ENCRYPT false
    CKA_VERIFY true
    CKA_VERIFY_RECOVER false
    CKA_WRAP false
    CKA_TRUSTED false
    CKA_EC_PARAMS length 10 (80 bits)
      { 0x06, 0x08, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07 }
    CKA_EC_POINT length 67 (536 bits)
      { 0x04, 0x41, 0x04, 0xBE, 0xDC, 0xBD, 0x5D, 0x46, 0x94, 0xC5, 0x0A, 0xF0,
        0x3E, 0xF3, 0x84, 0x74, 0x52, 0x62, 0x58, 0x4A, 0x60, 0xE0, 0x3B, 0x00,
        0xF4, 0xF9, 0xF6, 0x72, 0x98, 0x59, 0xF2, 0xAB, 0xD5, 0x8C, 0xD3, 0x99,
        0xBC, 0x81, 0x3E, 0xE1, 0xBD, 0xDC, 0x71, 0x0E, 0xCF, 0x07, 0xCE, 0xF2,
        0x92, 0xEF, 0x0E, 0x9E, 0x1D, 0x66, 0x0C, 0xC3, 0x93, 0xDB, 0x34, 0x7F,
        0x7F, 0xB3, 0x05, 0x7B, 0xE4, 0xEE, 0x2E }

## Rotating Keys

Unlike `notary init`, `notary key rotate` is capable of generating HSM-protected keys.

### Rotating the root key

    $ notary key rotate -K pkcs11 -T label:ocs1 -v example.com/collection root
    Warning: you are about to rotate your root key.

    You must use your old key to sign this root rotation.
    Are you sure you want to proceed?  (yes/no)
    Enter the passphrase for the PKCS#11 token 'nCipher Corp. Ltd ocs1 (7e585d361027d0e6)':
    Enter passphrase for snapshot key with ID 783c289:
    Successfully rotated root key for repository example.com/nshield

### Rotating the targets key

    $ notary key rotate -K pkcs11 -T label:ocs1 -v example.com/nshield targets
    Enter the passphrase for the PKCS#11 token 'nCipher Corp. Ltd ocs1 (7e585d361027d0e6)':
    Successfully rotated targets key for repository example.com/nshield

## Removing Keys

    $ notary key remove ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca
    
    Are you sure you want to remove ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca (role root) from pkcs11?  (yes/no)  yes
    Enter the passphrase for the PKCS#11 token 'nCipher Corp. Ltd ocs1 (7e585d361027d0e6)':
    
    Deleted ea90ae746cbb336030db3e8715218ce4dcd0411e65c37c80bf78959ff1774dca (role root) from pkcs11.

You can also remove it with the HSM's native tools
if you can correctly identify it.

# Developer Information

## Testing

Currently Notary's tests can fail if `NOTARY_HSM_LIB` is set,
because the expectation of an empty initial set of keys is violated
by any keys available on the HSM.

## Issues

* `notary key passwd` panics if you try to change the password
on a key held in the `pkcs11` keystore.
The reason is that the `Private()` method lacks any other way to signal an error.
* `notary key export` does not export keys from the `pkcs11` keystore.
This is consistent with its description as exporting from “local keystores”
but may be confusing nevertheless.
