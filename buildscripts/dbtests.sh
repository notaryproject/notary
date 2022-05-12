#!/usr/bin/env bash

db="$1"

case ${db} in
  mysql*)
    db="mysql"
    dbContainerOpts="mysql mysqld --innodb_file_per_table"
    DBURL="server@tcp(mysql:3306)/notaryserver?parseTime=True"
    ;;
  rethink*)
    db="rethink"
    dbContainerOpts="rdb-01 --bind all --driver-tls-key /tls/key.pem --driver-tls-cert /tls/cert.pem"
    DBURL="rdb-01.rdb"
    ;;
  postgresql*)
    db="postgresql"
    dbContainerOpts="postgresql -l"
    DBURL="postgres://server@postgresql:5432/notaryserver?sslmode=verify-ca&sslrootcert=/go/src/github.com/theupdateframework/notary/fixtures/database/ca.pem&sslcert=/go/src/github.com/theupdateframework/notary/fixtures/database/notary-server.pem&sslkey=/go/src/github.com/theupdateframework/notary/fixtures/database/notary-server-key.pem"
    ;;
  *)
    echo "Usage: $0 (mysql|rethink|postgresql)"
    exit 1
    ;;
esac

composeFile="development.${db}.yml"
project=dbtests

function cleanup {
    rm -f bin/notary
    docker-compose -p "${project}_${db}" -f "${composeFile}" kill ||:
    # if we're in CircleCI, we cannot remove any containers
    if [[ -z "${CIRCLECI}" ]]; then
        docker-compose -p "${project}_${db}" -f "${composeFile}" down -v --remove-orphans ||:
    fi
}

clientCmd="make TESTOPTS='-p 1' test"
if [[ -z "${CIRCLECI}" ]]; then
    BUILDOPTS="--force-rm"
else
    clientCmd="make ci && codecov"
fi

set -e
set -x

cleanup

docker-compose -p "${project}_${db}" -f ${composeFile} build ${BUILDOPTS} client

trap cleanup SIGINT SIGTERM EXIT

# run the unit tests that require a DB

docker-compose -p "${project}_${db}" -f "${composeFile}" run --no-deps --use-aliases -d ${dbContainerOpts}
docker-compose -p "${project}_${db}" -f "${composeFile}" run --no-deps \
    -e NOTARY_BUILDTAGS="${db}db" -e DBURL="${DBURL}" \
    -e PKGS="github.com/theupdateframework/notary/server/storage github.com/theupdateframework/notary/signer/keydbstore" \
    client bash -c "${clientCmd}"
