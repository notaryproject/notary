#!/usr/bin/env bash

#   Copyright The containerd Authors.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.


set -eu -o pipefail

go mod tidy
if [ -d vendor ]; then
  rm -rf vendor/
  go mod vendor
fi

DIFF_PATH="vendor/ go.mod go.sum"

# need word splitting here to avoid reading the whole DIFF_PATH as one pathspec
#
# shellcheck disable=SC2046
DIFF=$(git status --porcelain -- $DIFF_PATH)

if [ "$DIFF" ]; then
    echo
    echo "These files were modified:"
    echo
    echo "$DIFF"
    echo
    exit 1
else
    echo "$DIFF_PATH is correct"
fi
