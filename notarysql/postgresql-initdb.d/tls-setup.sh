#!/bin/bash

# Setup the server so it knows where to find certs so that server can be
# started with TLS enabled.
set -e

cp /docker-entrypoint-initdb.d/pg_hba.conf "$PGDATA"
cp /docker-entrypoint-initdb.d/server.{crt,key} "$PGDATA"
cp /docker-entrypoint-initdb.d/root.crt "$PGDATA"
chown postgres:postgres "$PGDATA"/server.{crt,key}
chown postgres:postgres "$PGDATA"/root.crt
chmod 0600 "$PGDATA"/server.key
