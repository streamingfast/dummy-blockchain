#!/bin/bash
# Copyright 2021 dfuse Platform Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

ACME_ROOT=${ACME_ROOT:-""}
if [[ ! -d "$ACME_ROOT" ]]; then
  echo "To generate 'dummy-blockchain' types correctly, you need to define environment"
  echo "variable 'ACME_ROOT' and making it point to 'firehose-acme' root directory. It's right"
  echo "now '$ACME_ROOT' which is either undefined or pointing to a non-existent location."
  exit 1
fi

# Protobuf definitions
PROTO_ACME=${1:-"$ACME_ROOT/proto"}

function main() {
  checks

  set -e
  cd "$ROOT/pb" &> /dev/null

  generate "sf/acme/type/v1/type.proto"

  echo "generate.sh - `date` - `whoami`" > ./last_generate.txt
  echo "firehose-acme/proto revision: `GIT_DIR=$ACME_ROOT/.git git log -n 1 --pretty=format:%h -- proto`" >> ./last_generate.txt
}

# usage:
# - generate <protoPath>
# - generate <protoBasePath/> [<file.proto> ...]
function generate() {
    base=""
    if [[ "$#" -gt 1 ]]; then
      base="$1"; shift
    fi

    for file in "$@"; do
      protoc "-I$PROTO_ACME" \
        --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false \
         $base$file
    done
}

function checks() {
  # The old `protoc-gen-go` did not accept any flags. Just using `protoc-gen-go --version` in this
  # version waits forever. So we pipe some wrong input to make it exit fast. This in the new version
  # which supports `--version` correctly print the version anyway and discard the standard input
  # so it's good with both version.
  result=`printf "" | protoc-gen-go --version 2>&1 | grep -Eo "v1.(2[6-9]|[3-9][0-9]+)"`
  if [[ "$result" == "" ]]; then
    echo "Your version of 'protoc-gen-go' (at `which protoc-gen-go`) is not recent enough."
    echo ""
    echo "To fix your problem, perform those commands:"
    echo ""
    echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@vlatest"
    echo ""
    echo "If everything is working as expetcted, the command:"
    echo ""
    echo "  protoc-gen-go --version"
    echo ""
    echo "Should print 'protoc-gen-go v1.32.0' (if it just hangs, you don't have the correct version)"
    exit 1
  fi
}

main "$@"