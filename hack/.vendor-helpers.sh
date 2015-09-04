#!/usr/bin/env bash

PROJECT=github.com/docker/notary

# Downloads dependencies into vendor/ directory
mkdir -p vendor

rm -rf .gopath
mkdir -p .gopath/src/github.com/docker
ln -sf ../../../.. .gopath/src/${PROJECT}
export GOPATH="${PWD}/.gopath:${PWD}/vendor"

clone() {
	local vcs="$1"
	local pkg="$2"
	local rev="$3"
	local url="$4"

	: ${url:=https://$pkg}
	local target="vendor/src/$pkg"

	echo -n "$pkg @ $rev: "

	if [ -d "$target" ]; then
		echo -n 'rm old, '
		rm -rf "$target"
	fi

	echo -n 'clone, '
	case "$vcs" in
		git)
			git clone --quiet --no-checkout "$url" "$target"
			( cd "$target" && git reset --quiet --hard "$rev" )
			;;
		hg)
			hg clone --quiet --updaterev "$rev" "$url" "$target"
			;;
	esac

	echo -n 'rm VCS, '
	( cd "$target" && rm -rf .{git,hg} )

	echo -n 'rm vendor, '
	( cd "$target" && rm -rf vendor Godeps/_workspace )

	echo done
}

# get an ENV from the Dockerfile with support for multiline values
_dockerfile_env() {
	local e="$1"
	awk '
		$1 == "ENV" && $2 == "'"$e"'" {
			sub(/^ENV +([^ ]+) +/, "");
			inEnv = 1;
		}
		inEnv {
			if (sub(/\\$/, "")) {
				printf "%s", $0;
				next;
			}
			print;
			exit;
		}
	' Dockerfile
}

clean() {
	local packages=(
		${PROJECT}/cmd/notary # cmd notary
		${PROJECT}/cmd/notary-server # cmd notary-server
		${PROJECT}/cmd/notary-signer # cmd notary-signer
	)

	local notaryPlatforms=( linux/amd64 $(_dockerfile_env NOTARY_CROSSPLATFORMS) )
	local notaryBuildTags="$(_dockerfile_env DOCKER_BUILDTAGS)"
	local buildTagCombos=(
		''
		"$notaryBuildTags"
	)

	echo

	echo -n 'collecting import graph, '
	local IFS=$'\n'
	packages+=( $(
		for platform in "${notaryPlatforms[@]}"; do
			export GOOS="${platform%/*}";
			export GOARCH="${platform##*/}";
			for buildTags in "${buildTagCombos[@]}"; do
				go list -e -tags "$buildTags" -f '{{join .Deps "\n"}}' "${packages[@]}"
			done
		done | grep -E "^${PROJECT}" | sort -u
	) )
	local imports=( $(
		for platform in "${notaryPlatforms[@]}"; do
			export GOOS="${platform%/*}";
			export GOARCH="${platform##*/}";
			for buildTags in "${buildTagCombos[@]}"; do
				go list -e -tags "$buildTags" -f '{{join .Deps "\n"}}' "${packages[@]}"
				go list -e -tags "$buildTags" -f '{{join .TestImports "\n"}}' "${packages[@]}"
			done
		done | grep -vE "^${PROJECT}" | sort -u
	) )
	imports=( $(go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' "${imports[@]}") )
	unset IFS

	echo -n 'pruning unused packages, '
	findArgs=(
		# This directory contains only .c and .h files which are necessary
		-path vendor/src/github.com/mattn/go-sqlite3/code
		-or
		-path vendor/src/code.google.com/p/gosqlite/sqlite3
		-or
		-path vendor/src/github.com/google/gofuzz
	)
	for import in "${imports[@]}"; do
		[ "${#findArgs[@]}" -eq 0 ] || findArgs+=( -or )
		findArgs+=( -path "vendor/src/$import" )
	done
	local IFS=$'\n'
	local prune=( $(find vendor -depth -type d -not '(' "${findArgs[@]}" ')') )
	unset IFS
	for dir in "${prune[@]}"; do
		find "$dir" -maxdepth 1 -not -type d -not -name 'LICENSE*' -not -name 'COPYING*' -exec rm -v -f '{}' +
		rmdir "$dir" 2>/dev/null || true
	done

	echo -n 'pruning unused files, '
	find vendor -type f -name '*_test.go' -exec rm -v '{}' +

	echo done
}
