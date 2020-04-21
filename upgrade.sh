#!/bin/bash

# git reset --hard HEAD
# git pull
git fetch --all && git reset --hard origin/master && git pull
go build -v -ldflags "-X github.com/lengzhao/govm/conf.BuildTime=$(date +%Y-%m%d-%H:%M) -X github.com/lengzhao/govm/conf.GitHead=$(git rev-parse HEAD)"
