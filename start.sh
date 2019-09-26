#!/bin/bash
while true
do
    echo start govm
    date
    ./govm
    echo something wrong, govm exit. $?
done