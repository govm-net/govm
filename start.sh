#!/bin/bash

file="conf/conf.json"
if [ ! -e "$file" ]
then
    echo not exist the config file
    cp "conf/conf.json.bak" $file
fi

echo start govm. you can use \"Ctrl + c\" to exit
date
./govm
rst=$?
if [ "$rst" != "0" ]
then
    echo "error",$rst
fi
echo "Enter to exit"  
read kkk
exit 0