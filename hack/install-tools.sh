#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

GO_MINOR_VERSION=$(go env GOVERSION | cut -c 2- | cut -d' ' -f1 | cut -d'.' -f2)
function go::install() {
  set -e
  GO_INSTALL_TMP_DIR=$(mktemp -d)
  cd "$GO_INSTALL_TMP_DIR"
  go mod init tmp
  if [ "$GO_MINOR_VERSION" -gt "16" ]; then
    GO111MODULE=on go install "$@"
  else
    GO111MODULE=on go get "$@"
  fi
  rm -rf "$GO_INSTALL_TMP_DIR"
}

go::install "$@"
