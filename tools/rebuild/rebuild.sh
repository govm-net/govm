#!/bin/bash

go build -o rebuild.exe
cd ../../
./tools/rebuild/rebuild.exe
