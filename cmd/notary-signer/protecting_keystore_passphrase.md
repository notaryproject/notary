Proposal: Protecting keystore passphrase 
==========================================


TL;DR Passphrases used to wrap private keys and store in the keystore DB, also need to be protected.

----------

Table of Contents
----------
[TOC]



Background
-------------

Currently in Nortary Signer, the signing keys are maintained in memory or are encrypted using AES wrapping with a password and the cipher text is persisted in a MySQL database. 

However, the password used to AES wrap the private keys is as an environment variable. 

If using the MySQL DB as a key store to persist the wrapped private keys, then the storage configruation in the signer configuration file looks as follows:

```
"storage": {
	"backend": "mysql",
	"db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true",
	"default_alias": "passwordalias1"
}

```
 - In cmd/notary-signer/config.go the default_alias field is concatenated with an  envPrefix string 'NOTARY_SIGNER\_' and the resulting string  '	NOTARY_SIGNER_<DEFAULT_ALIAS_VALUE>' gives the name of the environment variable that Notary signer looks for in the getEnv function.
 - The value of this environment variable contains the passphrase used to wrap the signing private keys and store in a key store like MySQL database, etc.   


Problem Statement
-------------
With the current method of protecting private keys, the private keys are persisted as AES wrapped cipher text, but the passphrase used to wrap the private key is still available in clear text in the environemt. The problem with this method is that the private keys on the signer are as secure as the environment variable. Infact, the whole signer is as secure as this environment variable.



Solution
-------------
As we see, there is a need to protect the passphrase. There are several options, but firstly, we need to start with an additional field in the storage configuration to configure the passphrase protection method. Please find below, the same storage configuration example with an additional field 'passphrase_protect_method'.

```
"storage": {
	"backend": "mysql",
	"db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true",
	"default_alias": "",
	"passphrase_protect_method" : "<none/hsm/other future methods>"
}

```

###Options for protection
 - None. This is the enviroment variable way of storing the passphrase. This will be default if nothing else is specified. 
 - HSMs. Hardware security modules have been time tested to be better at safeguarding sensitive data like passphrases from disclosure and from tampering. This would mean integrating with a PKCS11 interface. Additional configuration specific to PKCS11 needs to be configured.
 - Existing open source vault solutions like Vault, KeyWhiz, etc.
 - Other future solutions.

### HSM solution
PKCS11 is a standard interface to integrate with a wide variety of cryptographic tokens. However, procuring hardware is expensive and also involves more steps to install if not using an a cloud based HSM service. So, PKCS11 needs to be made optional and for only users who would like to take extra measures to safeguard the passphrase.

###Proposed changes in code
Callback functions as follows will be implemented for each type of protection mechanism:

```
type initPasswordProtect(protectMethod string)


type protectPassword func(alias string, createNew bool, password string) (status bool, err error)


type retrievePassword func(alias string) (password string, err error)

```

 - In the command we need to have additional options to store password.
 - This will be called back in passphraseRetriever function in cmd/notary-signer/config.go to retrieve the password.
 
 


###Proposed Flow

![Proposed flow](https://github.com/rhonnava/notary_proposal/blob/master/NewFlow.png)



