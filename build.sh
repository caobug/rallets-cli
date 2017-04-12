#!/bin/bash

version=`grep 'const VERSION = ' $(dirname $0)/const.go | sed -e 's/.*= //' | sed -e 's/"//g'`
echo "Creating rallets-cli $version"

build() {
    local name
    local GOOS
    local GOARCH

    prog=rallets-cli-$1-$2
    echo "Building $prog"
    GOOS=$1 GOARCH=$2 go build -o $prog rallets-cli || exit 1
}

# build darwin amd64
build linux amd64
build linux 386
# build windows amd64
# build windows 386
