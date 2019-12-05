#!/bin/bash

while true
do
    echo start govm. you can use "Ctrl + c" to exit
    date
    ./govm
    if [ "$?" -eq "127" ]
    then
        echo "stop by user(Ctrl + c)"
        exit $?
    fi
    if [ "$?" -eq "0" ]
    then
        echo "normal exit by user"
        exit $?
    fi
    echo something wrong, govm exit. $?
done