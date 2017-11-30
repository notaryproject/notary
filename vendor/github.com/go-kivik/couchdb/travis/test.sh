#!/bin/bash
set -euC

if [ "${TRAVIS_OS_NAME:-}" == "osx" ]; then
    # We don't have docker in OSX, so skip these tests
    unset KIVIK_TEST_DSN_COUCH16
    unset KIVIK_TEST_DSN_COUCH20
fi

function join_list {
    local IFS=","
    echo "$*"
}

case "$1" in
    "standard")
        go test -race $(go list ./... | grep -v /vendor/)
    ;;
    "linter")
        gometalinter.v1 --config .linter_test.json ./...
        gometalinter.v1 --config .linter.json ./...
    ;;
    "coverage")
        echo "" > coverage.txt

        # Filter /test/ to not count integration tests in coverage reports
        TEST_PKGS=$(find -name "*_test.go" | grep -v /vendor/ | grep -v /test/ | xargs dirname | sort -u | sed -e "s#^\.#github.com/go-kivik/couchdb#" )

        for d in $TEST_PKGS; do
            go test -i $d
            DEPS=$((go list -f $'{{range $f := .TestImports}}{{$f}}\n{{end}}{{range $f := .Imports}}{{$f}}\n{{end}}' $d && echo $d) | sort -u | grep -v /vendor/ | grep ^github.com/go-kivik/couchdb | tr '\n' ' ')
            go test -coverprofile=profile.out -covermode=set -coverpkg=$(join_list $DEPS) $d
            if [ -f profile.out ]; then
                cat profile.out >> coverage.txt
                rm profile.out
            fi
        done

        bash <(curl -s https://codecov.io/bash)
esac
