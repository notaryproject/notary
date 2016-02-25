<!--[metadata]>
+++
title = "Running a Notary Service"
description = "Run your own notary service to host arbitrary content signing."
keywords = ["docker, notary, notary-server, notary server, notary-signer, notary signer"]
[menu.main]
parent="mn_notary"
weight=4
+++
<![end-metadata]-->

This document assumes a familiarity with using
[Docker](https://docs.docker.com/engine/userguide/) and
[Docker Compose](https://docs.docker.com/compose/overview/).

<a name="top"></a>

- [Running a service for testing or development](#notary-service-temp)
- [Advanced configuration options](#notary-service-configuration)
- [Recommendations for deploying a production service](#notary-service-prod)
- [Recommendations for a high availability](#notary-service-prod)

### Running a service for testing or development <a name="notary-service-temp"></a>

The quickest way to spin up a full Notary service for testing and development
purposes is to use the Docker compose file in the
[Notary project](https://github.com/docker/notary).

```plain
$ git clone https://github.com/docker/notary.git
$ cd notary
$ docker-compose up
```

This will build the development Notary Server and Notary Signer images, and
start up a containers for the Notary Server, Notary Signer, and the MySQL
database that both of them share.  The MySQL data is stored in a

Notary Server and Notary Signer communicate over mutually authenticated TLS
(using the self-signed testing certs in the repository), and Notary Server
listens for HTTPS traffic on port 4443.

By default, this development Notary Server container runs with the testing
self-signed TLS certificates. In order to be able to successfully connect to
it, you will have to use the root CA file in `fixtures/root-ca.crt`.

For example, to connect using OpenSSL:

```plain
$ openssl s_client -connect <docker host>:4443 -CAfile fixtures/root-ca.crt -no_ssl3 -no_ssl2
```

To connect using the Notary Client CLI, please see [Getting Started](getting_started.md)
documentation.

The self-signed certificate's subject name and subject alternative names are
`notary-server`, `notaryserver`, and `localhost`, so if your Docker host not
on localhost (for example if you are using Docker Machine), you'll need to
update your hosts file such that the name `notary-server` is associated with
the IP address of your Docker host.

[[back to top](#top)]

### Advanced configuration options <a name="notary-service-configuration"></a>

Both the Notary Server and the Notary Signer take
[JSON configuration files](configuration.md). Pre-built images, such as
the [development images above](#notary-service-temp) provide these configuration
files for you with some sane defaults.

However, for running in production, or if you just want to change those defaults
on your development service, you probably want to change those defaults.

- **Running with different command line arguments**

	You can override the `docker run` command for the image if you want to pass
	different command line options.  Both Notary Server and Notary Signer take
	the following command line arguments:

	- `-config=<config file>` - specify the path to the JSON configuration file.

	- `-debug` - Passing this flag enables the debugging server on `localhost:8080`.
		The debugging server provides [pprof](https://golang.org/pkg/net/http/pprof/)
		and [expvar](https://golang.org/pkg/expvar/) endpoints.  (Remember, this
		is localhost with respect to the running container - this endpoint is not
		exposed from the container).

		This option can also be set in the configuration file.

	- `-logf=<format>` - This flag sets the output format for the logs. Possible
	    formats are "json" and "logfmt".

	    This option cannot be set in the configuration file, since some log
	    messages are produced on startup before the configuration file has been
	    read.

- **Specifying your own configuration files**

	You can run the images with your own configuration files entirely.
	You just need to mount your configuration directory, and then pass the
	path to that configuration file as an argument to the `docker run` command.

- **Overriding configuration file parameters using environment variables**

	You can also override the parameters of the configuration by
	setting environment variables of the form `NOTARY_SERVER_<var>` or
	`NOTARY_SIGNER_<var>`.

	`var` is the ALL-CAPS, `"_"`-delimited path of keys from the top level of the
	configuration JSON.

	For instance, if you wanted to override the storage URL of the Notary Server
	configuration:

	```json
	"storage": {
	  "backend": "mysql",
	  "db_url": "dockercondemo:dockercondemo@tcp(notary-mysql)/dockercondemo"
	}
	```

	You would need to set the environment variable `NOTARY_SERVER_STORAGE_DB_URL`,
	because the `db_url` is in the `storage` section of the Notary Server
	configuration JSON.

	Note that you cannot override a key whose value is another map.
	For instance, setting
	`NOTARY_SERVER_STORAGE='{"storage": {"backend": "memory"}}'` will not
	set in-memory storage.  It just fails to parse.  You can only override keys
	whose values are strings or numbers.

For example, let's say that you wanted to run a single Notary Server instance:

- with your own TLS cert and keys
- with a local, in-memory signer service rather than using Notary Signer,
- using a local, in-memory TUF metadata store rather than using MySQL
- produce JSON-formatted logs

One way to do this would be:

1. Generate your own TLS certificate and key as `server.crt` and `server.key`,
	and put them in the directory `/tmp/server-configdir`.

2. Write the following configuration file to `/tmp/server-configdir/config.json`:

	```json
	{
	  "server": {
	    "http_addr": ":4443",
	    "tls_key_file": "./server.key",
		"tls_cert_file": "./server.crt"
	  },
	  "trust_service": {
	    "type": "remote",
	    "hostname": "notarysigner",
	    "port": "7899",
	    "tls_ca_file": "./root-ca.crt",
	    "key_algorithm": "ecdsa",
	    "tls_client_cert": "./notary-server.crt",
	    "tls_client_key": "./notary-server.key"
	  },
	  "storage": {
	    "backend": "mysql",
	    "db_url": "server@tcp(mysql:3306)/notaryserver?parseTime=True"
	  }
	}
	```

	Note that we are including a remote trust service and a database storage
	type in order to demonstrate how environment variables can override
	configuration parameters.

3. Run the following command (assuming you've already built or pulled a
	Notary Server docker image):

	```plain
	$ docker run \
		-p "4443:4443" \
		-v /tmp/server-configdir:/etc/docker/notary-server/ \
		-e NOTARY_SERVER_TRUST_SERVICE_TYPE=local \
		-e NOTARY_SERVER_STORAGE_BACKEND=memory \
		-e NOTARY_SERVER_LOGGING_LEVEL=debug \
		notary_server \
			-config=/etc/docker/notary-server/config.json \
			-logf=json
	{"level":"info","msg":"Version: 0.2, Git commit: 619f8cf","time":"2016-02-25T00:53:59Z"}
	{"level":"info","msg":"Using local signing service, which requires ED25519. Ignoring all other trust_service parameters, including keyAlgorithm","time":"2016-02-25T00:53:59Z"}
	{"level":"info","msg":"Using memory backend","time":"2016-02-25T00:53:59Z"}
	{"level":"info","msg":"Starting Server","time":"2016-02-25T00:53:59Z"}
	{"level":"info","msg":"Enabling TLS","time":"2016-02-25T00:53:59Z"}
	{"level":"info","msg":"Starting on :4443","time":"2016-02-25T00:53:59Z"}
	```

You can do the same using
[Docker Compose](https://docs.docker.com/compose/overview/) by setting volumes,
environment variables, and overriding the default command for the Notary Server
containers in the compose file.

[[back to top](#top)]

### Recommendations for deploying a production service <a name="notary-service-prod"></a>
### Recommendations for a high availability <a name="notary-service-ha"></a>

![Notary Server Deployment Diagram](service-deployment.svg)
