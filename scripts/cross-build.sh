#!/bin/bash
export GOPATH=$HOME/go

cd "$(dirname ${BASH_SOURCE[0]})"

set -x

mkdir -p builds/tmp
cd builds/tmp

version=0.0.2
githash=$(git rev-parse HEAD)

/opt/bin/xgo -v -x  -ldflags "-X main.AppVersion=$version -X main.AppGitHash=$githash" --targets=darwin/amd64,windows/386 ../../../
mv gdax-bookmap-windows-4.0-386.exe gdax-bookmap.exe
zip -r ../gdax-bookmap-win32.zip gdax-bookmap.exe
rm -f gdax-bookmap.exe

cp -r ../../macOS-tmpl gdax-bookmap.app
cp gdax-bookmap-darwin-10.6-amd64 gdax-bookmap.app/Contents/MacOS/gdax-bookmap
zip -r ../gdax-bookmap-osx.zip gdax-bookmap.app
rm -rf gdax-bookmap.app gdax-bookmap-darwin-10.6-amd64

cd ..
echo VERSION gdax-bookmap $version-$githash > checksum.txt
sha256sum gdax-bookmap-win32.zip gdax-bookmap-osx.zip >> checksum.txt
