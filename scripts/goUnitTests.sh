#!/bin/bash
###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
###

set -e

export GO15VENDOREXPERIMENT=1
echo -n "Obtaining list of tests to run.."
PKGS=`go list github.com/maplerime/cl-query/... | grep -v /vendor/ | grep -v /examples/ | grep -v /mock/ | grep -v /utils/`
echo "DONE!"

echo "Running tests..."
echo $PKGS
go test -cover -p 1 -timeout=20m $PKGS
