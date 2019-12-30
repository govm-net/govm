#!/bin/bash

while true
do
    echo start govm. you can use "Ctrl + c" to exit
    date
    ./govm
    rst=$?
    if [ "$rst" -eq "127" ]
    then
        echo "stop by user(Ctrl + c)"
        exit $rst
    fi
    if [ "$rst" -eq "0" ]
    then
        echo "normal exit by user"
        exit $rst
    fi
    echo something wrong, govm exit. $rst
done
