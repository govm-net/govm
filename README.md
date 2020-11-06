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
7. browser open <http://localhost:9090>

## upgrade

1. stop govm
2. ./upgrade.sh
3. ./govm

## exchange

ERC20: wGOVM(Wrapped GOVM)

wGOVM: 0xaC5d7dFF150B195C97Fca77001f8AD596eda1761

uniswap: <https://app.uniswap.org/#/swap?outputCurrency=0xac5d7dff150b195c97fca77001f8ad596eda1761>

balancer: <https://balancer.exchange/#/swap?outputCurrency=0xac5d7dff150b195c97fca77001f8ad596eda1761>

wGOVM->govm:

1. Requires Metamask and eth
2. Chrome open:<http://govm.net:9090/wgovm.html>
3. Burn(wGOVM->govm):Approve(just first time)
4. input "Amount" and your "GOVM Address"
5. click "Burn"
6. change the gas price,then submit

## plan

see <http://govm.net>
