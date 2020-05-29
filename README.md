# govm

this is a blockchain system based multi-chain.
Theoretical TPS can exceed 2^64.

cn: 这是一个基于同构多链的区块链系统。它的理论性能可以超过2^64。

## Building the source

1. install golang(>v1.13),git
2. start database: github.com/lengzhao/database
3. download code
4. build: go build
5. change config:conf/conf.json
6. start software: ./govm
7. browser open http://localhost:9090

## upgrade

1. stop govm
2. ./upgrade.sh
3. ./govm

## plan

see http://govm.net
