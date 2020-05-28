#!/bin/bash

# git reset --hard HEAD
# git pull
git fetch --all && git reset --hard origin/master && git pull
go build -v -ldflags "-X github.com/govm-net/govm/conf.BuildTime=$(date +%Y-%m%d-%H:%M) -X github.com/govm-net/govm/conf.GitHead=$(git rev-parse HEAD)"
