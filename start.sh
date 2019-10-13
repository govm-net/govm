#!/bin/bash

git pull
go build -v -ldflags "-X github.com/lengzhao/govm/conf.BuildTime=$(date +%Y-%m%d-%H:%M) -X github.com/lengzhao/govm/conf.GitHead=$(git rev-parse HEAD | echo unknow)"

while true
do
    echo start govm
    date
    ./govm
    if [ "$?" -eq "127" ]
    then
        echo "stop by user"
        exit $?
    fi
    echo something wrong, govm exit. $?
done