#!/bin/bash
###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2024 All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
###

echo "Running golangci-lint in docker mode"

docker run -t --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.62.0 golangci-lint run -v --timeout 10m

echo "golangci-lint completed"
# End of script
