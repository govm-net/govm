#!/bin/bash

git reset --hard HEAD
git pull
go build -v -ldflags "-X github.com/lengzhao/govm/conf.BuildTime=$(date +%Y-%m%d-%H:%M) -X github.com/lengzhao/govm/conf.GitHead=$(git rev-parse HEAD)"

file="conf/conf.json"
if [ ! -e "$file" ]
then
    echo not exist the config file
    cp "conf/conf.json.bak" $file
fi

