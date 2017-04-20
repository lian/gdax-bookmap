#!/bin/bash
set -x

version=alpha-v5
githash=$(git rev-parse HEAD)

case "$OSTYPE" in
    linux*)
      echo building linux binary
      go build -o gdax-bookmap-linux -v \
        -ldflags "-X main.AppVersion=$version -X main.AppGitHash=$githash" \
        main.go
      ;;
    darwin*)
      echo building macOS binary
      go build -o gdax-bookmap-osx -v -ldflags -s \ 
        -ldflags "-X main.AppVersion=$version -X main.AppGitHash=$githash" \
        main.go
      ;;
    msys)
      echo building Windows binary
      go build -o gdax-bookmap-win32.exe -v \
        -ldflags "-X main.AppVersion=$version -X main.AppGitHash=$githash" \
        main.go
      ;;
esac
