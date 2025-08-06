#!/bin/bash

###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2024 All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
###

#set -eux

PROTO_ROOT_DIR="$GOPATH/src/raksmart.com/rak_api/protos"
PROTO_DIRS=`find $GOPATH/src/raksmart.com/rak_api/protos -mindepth 1 -maxdepth 1 -type d`

for dir in $PROTO_DIRS; do
    echo Working on dir $dir
    protoc --proto_path="$PROTO_ROOT_DIR" --go_out=plugins=grpc:$GOPATH/src "$dir"/*.proto
done
