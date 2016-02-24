<!--[metadata]>
+++
title = "Notary Configuration"
description = "Configuring the notary client, server and signer."
keywords = ["docker, notary, notary-client, notary-server, notary server, notary-signer, notary signer"]
[menu.main]
parent="mn_notary"
weight=5
+++
<![end-metadata]-->

# Configuration files <a name="top"></a>

- [Notary Client](#notary-client-configuration-file)
- [Notary Server](#notary-server-configuration-file)
- [Notary Signer](#notary-signer-configuration-file)

## Notary Client Configuration File

The configuration file for Notary Client consists of the following sections:

<table>
	<tr>
		<td><a href="#notary-client-trustdir">trust_dir</a></td>
		<td>TUF key and metadata directory</td>
		<td>(optional)</td>
	</tr>
	<tr>
		<td><a href="#notary-client-remote">remote_server</a></td>
		<td>remote Notary Server configuration</td>
		<td>(optional)</td>
	</tr>
</table>

In addition please see the optional
[password environment variables](#notary-client-envvars) Notary Client can take
for ease of use.

An example (full) server configuration file.

```json
{
  "trust_dir" : "~/.docker/trust",
  "remote_server": {
    "url": "https://my-notary-server.my-private-registry.com",
    "root-ca": "./fixtures/root-ca.crt",
    "tls_client_cert": "./fixtures/secure.example.com.crt",
    "tls_client_key": "./fixtures/secure.example.com.crt"
  }
}
```

- #### `trust_dir` section (optional) <a name="notary-client-trustdir"></a>

	The `trust_dir` specifies the location (as an absolute path or a path
	relative to the directory of the configuration file) where the TUF metadata
	and private keys will be stored.

	This is normally defaults to `~/.notary`, but specifying `~/.docker/trust`
	facilitates interoperability with Docker Content Trust.

	[[Notary Client configuration overview](#notary-client-configuration-file)]

- #### `remote_server` section (optional) <a name="notary-client-remote"></a>

	The `remote_server` specifies how to connect to a Notary Server to download
	metadata updates and publish metadata changes.

	Remote server example:

	```json
	"remote_server": {
	  "url": "https://my-notary-server.my-private-registry.com",
	  "root-ca": "./fixtures/root-ca.crt",
	  "tls_client_cert": "./fixtures/secure.example.com.crt",
	  "tls_client_key": "./fixtures/secure.example.com.crt"
	}
	```

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>url</code></td>
			<td valign="top">no</td>
			<td valign="top">URL of the Notary Server: defaults to https://notary.docker.io</td>
		</tr>
		<tr>
			<td valign="top"><code>root-ca</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the file containing the root CA with which to verify
				the TLS certificate of the Notary Server, for example if it is self-signed.
				The path is relative to the directory of the configuration file.</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_client_cert</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the client certificate to use for mutual TLS with
				the Notary Server.  Must be provided along with <code>tls_client_key</code>
				or not provided at all.  The path is relative to the directory of the
				configuration file.  </td>
		</tr>
		<tr>
			<td valign="top"><code>tls_client_key</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the client key to use for mutual TLS with
				the Notary Server. Must be provided along with <code>tls_client_cert</code>
				or not provided at all.  The path is relative to the directory of the
				configuration file.</td>
		</tr>
	</table>

	[[Notary Client configuration overview](#notary-client-configuration-file)]

- #### Environment variables (optional) <a name="notary-client-envvars"></a>

	The following environment variables containing signing key passphrases can
	be used to facilitate Notary Client CLI interaction.  If provided, these
	passwords will be used initially to sign TUF metadata.  If the passphrase
	is incorrect, you will be prompted to enter the correct passphrase.


	| Environment Variable        | Description                             |
	| --------------------------- | --------------------------------------- |
	|`NOTARY_ROOT_PASSPHRASE`     | The root/offline key passphrase         |
	|`NOTARY_TARGETS_PASSPHRASE`  | The targets (an online) key passphrase  |
	|`NOTARY_SNAPSHOT_PASSPHRASE` | The snapshot (an online) key passphrase |

	[[Notary Client configuration overview](#notary-client-configuration-file)]


[[back to top](#top)]

## Notary Server Configuration File

The configuration file for Notary Server consists of the following sections:

<table>
	<tr>
		<td><a href="#notary-server-server">server</a></td>
		<td>HTTPS configuration</td>
		<td>(required)</td>
	</tr>
	<tr>
		<td><a href="#notary-server-trust-service">trust_service</a></td>
		<td>signing service configuration</td>
		<td>(required)</td>
	</tr>
	<tr>
		<td><a href="#notary-server-storage">storage</a></td>
		<td>TUF metadata storage configuration</td>
		<td>(required)</td>
	</tr>
	<tr>
		<td><a href="#notary-server-auth">auth</a></td>
		<td>server authentication configuration</td>
		<td>(optional)</td>
	</tr>
	<tr>
		<td><a href="#server-signer-logging">logging</a></td>
		<td>logging configuration</td>
		<td>(optional)</td>
	</tr>
	<tr>
		<td><a href="#server-signer-reporting">reporting</a></td>
		<td>ops/reporting configuration</td>
		<td>(optional)</td>
	</tr>
</table>

An example (full) server configuration file.

```json
{
  "server": {
    "http_addr": ":4443",
    "tls_key_file": "./fixtures/notary-server.key",
    "tls_cert_file": "./fixtures/notary-server.crt"
  },
  "trust_service": {
    "type": "remote",
    "hostname": "notarysigner",
    "port": "7899",
    "key_algorithm": "ecdsa",
    "tls_ca_file": "./fixtures/root-ca.crt",
    "tls_client_cert": "./fixtures/notary-server.crt",
    "tls_client_key": "./fixtures/notary-server.key"
  },
  "storage": {
    "backend": "mysql",
    "db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true"
  },
  "auth": {
    "type": "token",
    "options": {
      "realm": "https://auth.docker.io/token",
      "service": "notary-server",
      "issuer": "auth.docker.io",
      "rootcertbundle": "/path/to/auth.docker.io/cert"
    }
  },
  "logging": {
    "level": "debug"
  },
  "reporting": {
    "bugsnag": {
      "api_key": "c9d60ae4c7e70c4b6c4ebd3e8056d2b8",
      "release_stage": "production"
    }
  }
}
```

- #### `server` section (required) <a name="notary-server-server"></a>

	Example:

	```json
	"server": {
	  "http_addr": ":4443",
	  "tls_key_file": "./fixtures/notary-server.key",
	  "tls_cert_file": "./fixtures/notary-server.crt"
	}
	```

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>http_addr</code></td>
			<td valign="top">yes</td>
			<td valign="top">The TCP address (IP and port) to listen on.  Examples:
				<ul>
				<li><code>":4443"</code> means listen on port 4443 on all IPs (and
					hence all interfaces, such as those listed when you run
					<code>ifconfig</code>)</li>
				<li><code>"127.0.0.1:4443"</code> means listen on port 4443 on
					localhost only.  That means that the server will not be
					accessible except locally (via SSH tunnel, or just on a local
					terminal)</li>
				</ul>
			</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_key_file</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the private key to use for
				HTTPS.  Must be provided together with <code>tls_cert_file</code>,
				or not at all. If neither are provided, the server will use HTTP
				instead of HTTPS. The path is relative to the directory of the
				configuration file.</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_cert_file</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the certificate to use for HTTPS.
				Must be provided together with <code>tls_key_file</code>, or not
				at all. If neither are provided, the server will use HTTP instead
				of HTTPS. The path is relative to the directory of the
				configuration file.</td>
		</tr>
	</table>

	[[Notary Server configuration overview](#notary-server-configuration-file)]

- #### `trust service` section (required) <a name="notary-server-trust-service"></a>

	This section configures either a remote trust service, such as
	[Notary Signer](#notary-signer-configuration-file) or a local in-memory
	ED25519 trust service.

	Remote trust service example:

	```json
	"trust_service": {
	  "type": "remote",
	  "hostname": "notarysigner",
	  "port": "7899",
	  "key_algorithm": "ecdsa",
	  "tls_ca_file": "./fixtures/root-ca.crt",
	  "tls_client_cert": "./fixtures/notary-server.crt",
	  "tls_client_key": "./fixtures/notary-server.key"
	}
	```

	Local trust service example:

	```json
	"trust_service": {
	  "type": "local"
	}
	```

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>type</code></td>
			<td valign="top">yes</td>
			<td valign="top">Must be <code>"remote"</code> or <code>"local"</code></td>
		</tr>
		<tr>
			<td valign="top"><code>hostname</code></td>
			<td valign="top">yes if remote</td>
			<td valign="top">The hostname of the remote trust service</td>
		</tr>
		<tr>
			<td valign="top"><code>port</code></td>
			<td valign="top">yes if remote</td>
			<td valign="top">The GRPC port of the remote trust service</td>
		</tr>
		<tr>
			<td valign="top"><code>key_algorithm</code></td>
			<td valign="top">yes if remote</td>
			<td valign="top">Algorithm to use to generate keys stored on the
				signing service.  Valid values are <code>"ecdsa"</code>,
				<code>"rsa"</code>, and <code>"ed25519"</code>.</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_ca_file</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the root CA that signed the TLS
				certificate of the remote service. This parameter if said root
				CA is not in the system's default trust roots. The path is
				relative to the directory of the configuration file.</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_client_key</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the private key to use for TLS mutual
				authentication. This must be provided together with
				<code>tls_client_cert</code> or not at all. The path is relative
				to the directory of the configuration file.</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_client_cert</code></td>
			<td valign="top">no</td>
			<td valign="top">The path to the certificate to use for TLS mutual
				authentication. This must be provided together with
				<code>tls_client_key</code> or not at all. The path is relative
				to the directory of the configuration file.</td>
		</tr>
	</table>

	[[Notary Server configuration overview](#notary-server-configuration-file)]

- #### `storage` section (required) <a name="notary-server-storage"></a>

	The storage section specifies which storage backend the server should use to
	store TUF metadata.  Currently, we only support MySQL or an in-memory store.

	DB storage example:

	```json
	"storage": {
	  "backend": "mysql",
	  "db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true"
	}
	```

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>backend</code></td>
			<td valign="top">yes</td>
			<td valign="top">Must be <code>"mysql"</code> or <code>"memory"</code>.
				If <code>"memory"</code> is selected, the <code>db_url</code>
				is ignored.</td>
		</tr>
		<tr>
			<td valign="top"><code>db_url</code></td>
			<td valign="top">yes if not <code>memory</code></td>
			<td valign="top">The <a href="https://github.com/go-sql-driver/mysql">
				the Data Source Name used to access the DB.</a>
				(note: please include "parseTime=true" as part of the the DSN)</td>
		</tr>
	</table>

	[[Notary Server configuration overview](#notary-server-configuration-file)]

- #### `auth` section (optional) <a name="notary-server-auth"></a>

	This sections specifies the authentication options for the server.
	Currently, we only support token authentication.

	Example:

	```json
	"auth": {
	  "type": "token",
	  "options": {
	    "realm": "https://auth.docker.io",
	    "service": "notary-server",
	    "issuer": "auth.docker.io",
	    "rootcertbundle": "/path/to/auth.docker.io/cert"
	  }
	}
	```

	Note that this entire section is optional.  However, if you would like
	authentication for your server, then you need the required parameters below to
	configure it.

	**Token authentication:**

	This is an implementation of the same authentication used by
	[docker registry](https://github.com/docker/distribution).  (JWT token-based
	authentication post login.)

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>type</code></td>
			<td valign="top">yes</td>
			<td valign="top">Must be `"token"`; all other values will result in no
				authentication (and the rest of the parameters will be ignored)</td>
		</tr>
		<tr>
			<td valign="top"><code>options</code></td>
			<td valign="top">yes</td>
			<td valign="top">The options for token auth.  Please see
				<a href="https://github.com/docker/distribution/blob/master/docs/configuration.md#token">
				the registry token configuration documentation</a>
				for the parameter details.</td>
		</tr>
	</table>

	[[Notary Server configuration overview](#notary-server-configuration-file)]

---
[[back to top](#top)]


## Notary Signer Configuration File

Notary signer [requires environment variables](#notary-signer-envvars) in order to
encrypt private keys at rest.  It also takes a configuration file, which
consists of the following sections:

<table>
	<tr>
		<td><a href="#notary-signer-server">server</a></td>
		<td>HTTPS and GRPC configuration</td>
		<td>(required)</td>
	</tr>
	<tr>
		<td><a href="#notary-signer-storage">storage</a></td>
		<td>TUF metadata storage configuration</td>
		<td>(required)</td>
	</tr>
	<tr>
		<td><a href="#server-signer-logging">logging</a></td>
		<td>logging configuration</td>
		<td>(optional)</td>
	</tr>
	<tr>
		<td><a href="#server-signer-reporting">reporting</a></td>
		<td>ops/reporting configuration</td>
		<td>(optional)</td>
	</tr>
</table>
<table>
	<tr>
		<td colspan=2>
			<a href="#notary-signer-envvars">for encrypting private keys</a></td>
		<td>(required)</td>
	</tr>
</table>


An example (full) server configuration file.

```json
{
  "server": {
    "http_addr": ":4444",
    "grpc_addr": ":7899",
    "tls_cert_file": "./fixtures/notary-signer.crt",
    "tls_key_file": "./fixtures/notary-signer.key",
    "client_ca_file": "./fixtures/notary-server.crt"
  },
  "logging": {
    "level": 2
  },
  "storage": {
    "backend": "mysql",
    "db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true",
    "default_alias": "passwordalias1"
  },
  "reporting": {
    "bugsnag": {
      "api_key": "c9d60ae4c7e70c4b6c4ebd3e8056d2b8",
      "release_stage": "production"
    }
  }
}
```

- #### `server` section (required) <a name="notary-signer-server"></a>

	"server" in this case refers to Notary Signer's HTTP/GRPC server, not
	"Notary Server".

	Example:

	```json
	"server": {
	  "http_addr": ":4444",
	  "grpc_addr": ":7899",
	  "tls_cert_file": "./fixtures/notary-signer.crt",
	  "tls_key_file": "./fixtures/notary-signer.key",
	  "client_ca_file": "./fixtures/notary-server.crt"
	}
	```

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>http_addr</code></td>
			<td valign="top">yes</td>
			<td valign="top">The TCP address (IP and port) to listen for HTTP
				traffic on.  Examples:
				<ul>
				<li><code>":4444"</code> means listen on port 4444 on all IPs (and
					hence all interfaces, such as those listed when you run
					<code>ifconfig</code>)</li>
				<li><code>"127.0.0.1:4444"</code> means listen on port 4444 on
					localhost only.  That means that the server will not be
					accessible except locally (via SSH tunnel, or just on a local
					terminal)</li>
				</ul>
			</td>
		</tr>
		<tr>
			<td valign="top"><code>grpc_addr</code></td>
			<td valign="top">yes</td>
			<td valign="top">The TCP address (IP and port) to listen for GRPC
				traffic.  Examples:
				<ul>
				<li><code>":7899"</code> means listen on port 7899 on all IPs (and
					hence all interfaces, such as those listed when you run
					<code>ifconfig</code>)</li>
				<li><code>"127.0.0.1:7899"</code> means listen on port 7899 on
					localhost only.  That means that the server will not be
					accessible except locally (via SSH tunnel, or just on a local
					terminal)</li>
				</ul>
			</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_key_file</code></td>
			<td valign="top">yes</td>
			<td valign="top">The path to the private key to use for
				HTTPS. The path is relative to the directory of the
				configuration file.</td>
		</tr>
		<tr>
			<td valign="top"><code>tls_cert_file</code></td>
			<td valign="top">yes</td>
			<td valign="top">The path to the certificate to use for
				HTTPS. The path is relative to the directory of the
				configuration file.</td>
		</tr>
		<tr>
			<td valign="top"><code>client_ca_file</code></td>
			<td valign="top">no</td>
			<td valign="top">The root certificate to trust for
				mutual authentication. If provided, any clients connecting to
				Notary Signer will have to have a client certificate signed by
				this root. If not provided, mutual authentication will not be
				required. The path is relative to the directory of the
				configuration file.</td>
		</tr>
	</table>

	[[Notary Signer configuration overview](#notary-signer-configuration-file)]

- #### `storage` section (required) <a name="notary-signer-storage"></a>

	This is used to store encrypted priate keys.  We only support MySQL or an
	in-memory store, currently.

	Example:

	```json
	"storage": {
	  "backend": "mysql",
	  "db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true",
	  "default_alias": "passwordalias1"
	}
	```

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>backend</code></td>
			<td valign="top">yes</td>
			<td valign="top">Must be <code>"mysql"</code> or <code>"memory"</code>.
				If <code>"memory"</code> is selected, the <code>db_url</code>
				is ignored.</td>
		</tr>
		<tr>
			<td valign="top"><code>db_url</code></td>
			<td valign="top">yes if not <code>memory</code></td>
			<td valign="top">The <a href="https://github.com/go-sql-driver/mysql">
				the Data Source Name used to access the DB.</a>
				(note: please include "parseTime=true" as part of the the DSN)</td>
		</tr>
		<tr>
			<td valign="top"><code>default_alias</code></td>
			<td valign="top">yes if not <code>memory</code></td>
			<td valign="top">This parameter specifies the alias of the current
				password used to encrypt the private keys in the DB.  All new
				private keys will be encrypted using this password, which
				must also be provided as the environment variable
				<code>NOTARY_SIGNER_&lt;DEFAULT_ALIAS_VALUE&gt;</code>.
				Please see the <a href="#notary-signer-envvars">environment variable</a>
				section for more information.</td>
		</tr>
	</table>

	[[Notary Signer configuration overview](#notary-signer-configuration-file)]

- #### Environment variables (required if using MySQL) <a name="notary-signer-envvars"></a>

	Notary Signer stores the private keys in encrypted form.
	The alias of the passphrase used to encrypt the keys is also stored.  In order
	to encrypt the keys for storage and decrypt the keys for signing, the
	passphrase must be passed in as an environment variable.

	For example, the configuration above specifies the default password alias to be
	`passwordalias1`.

	If this configuration is used, then you must:

	`export NOTARY_SIGNER_PASSWORDALIAS1=mypassword`

	so that that Notary Signer knows to encrypt all keys with the passphrase
	"mypassword", and to decrypt any private key stored with password alias
	"passwordalias1" with the passphrase "mypassword".

	Older passwords may also be provided as environment variables.  For instance,
	let's say that you wanted to change the password that is used to create new
	keys (rotating the passphrase and re-encrypting all the private keys is not
	supported yet).

	You could change the config to look like:

	```json
	"storage": {
	  "backend": "mysql",
	  "db_url": "user:pass@tcp(notarymysql:3306)/databasename?parseTime=true",
	  "default_alias": "passwordalias2"
	}
	```

	Then you can set:

	```bash
	export NOTARY_SIGNER_PASSWORDALIAS1=mypassword
	export NOTARY_SIGNER_PASSWORDALIAS2=mynewfancypassword
	```

	That way, all new keys will be encrypted and decrypted using the passphrase
	"mynewfancypassword", but old keys that were encrypted using the passphrase
	"mypassword" can still be decrypted.

	The environment variables for the older passwords are optional, but Notary
	Signer will not be able to decrypt older keys if they are not provided, and
	attempts to sign data using those keys will fail.

	[[Notary Signer configuration overview](#notary-signer-configuration-file)]

---

### Configuration sections common to Notary Server and Notary Signer

The logging and bug reporting configuration options for both Notary Server and
Notary Signer have the same keys and format:

- #### `logging` section (optional) <a name="server-signer-logging"></a>

	The logging section sets the log level of the server.  If it is not provided
	or invalid, the signer defaults to an ERROR logging level.

	Example:

	```json
	"logging": {
	  "level": "debug"
	}
	```

	Note that this entire section is optional.  However, if you would like to
	specify a different log level, then you need the required parameters
	below to configure it.

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>level</code></td>
			<td valign="top">yes</td>
			<td valign="top">One of <code>"debug"</code>, <code>"info"</code>,
				<code>"warning"</code>, <code>"error"</code>, <code>"fatal"</code>,
				or <code>"panic"</code></td>
		</tr>
	</table>

	[[Notary Server configuration overview](#notary-server-configuration-file)]
	[[Notary Signer configuration overview](#notary-signer-configuration-file)]

- #### `reporting` section (optional) <a name="server-signer-reporting"></a>

	The reporting section contains any configuration for useful for running the
	service, such as reporting errors. Currently, we only support reporting errors
	to [Bugsnag](https://bugsnag.com).

	See [bugsnag-go](https://github.com/bugsnag/bugsnag-go/) for more information
	about these configuration parameters.

	```json
	"reporting": {
	  "bugsnag": {
	    "api_key": "c9d60ae4c7e70c4b6c4ebd3e8056d2b8",
	    "release_stage": "production"
	  }
	}
	```

	Note that this entire section is optional.  However, if you would like to
	report errors to Bugsnag, then you need to include a `bugsnag` subsection,
	along with the required parameters below, to configure it.

	**Bugsnag reporting:**

	<table>
		<tr>
			<th>Parameter</th>
			<th>Required</th>
			<th>Description</th>
		</tr>
		<tr>
			<td valign="top"><code>api_key</code></td>
			<td valign="top">yes</td>
			<td>The BugSnag API key to use to report errors.</td>
		</tr>
		<tr>
			<td valign="top"><code>release_stage</code></td>
			<td valign="top">yes</td>
			<td>The current release stage, such as "production".  You can
				use this value to filter errors in the Bugsnag dashboard.</td>
		</tr>
	</table>

	[[Notary Server configuration overview](#notary-server-configuration-file)]
	[[Notary Signer configuration overview](#notary-signer-configuration-file)]
