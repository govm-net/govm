#!/bin/bash

go build -o rebuild.exe
cd `dirname "$0"`
cd ../../
./tools/rebuild/rebuild.exe
