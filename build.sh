#!/bin/bash

set -uo pipefail

TARGET=raascs
VER="$(git describe --tags --always --dirty | tr '-' '.')"

rm -rf pkgroot output
mkdir -p output pkgroot/{bin,logs,conf}

unset GOPATH
export GO111MODULE=on
export GOOS=linux
export GOARCH=amd64 
go build -ldflags "-X main.versionStr=$VER " -o "pkgroot/bin/$TARGET" main.go
cp -a status.sh status_functions.sh pkgroot/bin/
cp -a conf/* pkgroot/conf/

cd pkgroot
find . -type f | grep -Fv MD5.list | xargs md5sum >> MD5.list
tar -zcvf ../output/"$TARGET-$VER.tar.gz" bin logs conf MD5.list
cd -
