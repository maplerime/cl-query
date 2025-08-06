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

for i in "cmd pkg utils"
do
    OUTPUT="$(goimports -l -e $i)"
    if [[ $OUTPUT ]]; then
        echo "The following files contain goimports errors"
        echo $OUTPUT
        #goimports $OUTPUT > $OUTPUT
        for j in $OUTPUT
        do
          echo $j
          goimports $j > "txt"
          cat "txt" > $j
        done
        echo "The goimports command must be run for these files"
        #exit 1
    fi
    ./script/goimports
done
