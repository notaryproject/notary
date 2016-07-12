#!/usr/bin/env bash

composeFile="$1"

function cleanup {
    rm -f bin/notary
	docker-compose -f $composeFile kill
	# if we're in CircleCI, we cannot remove any containers
	if [[ -z "${CIRCLECI}" ]]; then
		docker-compose -f $composeFile down -v --remove-orphans
	fi
}

function cleanupAndExit {
    cleanup
    # Check for existence of SUCCESS
    ls test_output/SUCCESS
    exitCode=$?
    # Clean up test_output dir (if not in CircleCI) and exit
    if [[ -z "${CIRCLECI}" ]]; then
        rm -rf test_output
    fi
    exit $exitCode
}

if [[ -z "${CIRCLECI}" ]]; then
	BUILDOPTS="--force-rm"
fi

set -e
set -x

# cleanup

docker-compose -f $composeFile config
docker-compose -f $composeFile build ${BUILDOPTS} --pull | tee

trap cleanupAndExit SIGINT SIGTERM EXIT

# run the unit tests that require a DB
case $composeFile in
  development.mysql.yml)
    docker-compose -f $composeFile run --no-deps -d --name "mysql_tests" mysql mysqld --innodb_file_per_table
    docker-compose -f $composeFile run --no-deps \
        -e NOTARY_BUILDTAGS=mysqldb \
        -e PKGS="github.com/docker/notary/server/storage" \
        -e MYSQL="server@tcp(mysql_tests:3306)/notaryserver?parseTime=True" \
        --user notary \
        client bash -c "make ci && codecov"
    ;;
  development.rethink.yml)
    docker-compose -f $composeFile run --no-deps -d --name "rethinkdb_tests" \
        rdb-01 --bind all --no-http-admin --server-name rdb_01 \
        --driver-tls-key /tls/key.pem --driver-tls-cert /tls/cert.pem
    docker-compose -f $composeFile run --no-deps \
        -e NOTARY_BUILDTAGS=rethinkdb \
        -e PKGS="github.com/docker/notary/server/storage" \
        -e RETHINK="rethinkdb_tests" \
        --user notary \
        client bash -c "make ci && codecov"
    ;;
esac

docker-compose -f $composeFile down -v

# docker-compose -f $composeFile up --abort-on-container-exit
