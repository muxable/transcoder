#!/bin/bash

docker run -v $PWD:/defs namely/protoc-all -f transcoder.proto -l go
