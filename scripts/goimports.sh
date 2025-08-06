#!/bin/bash
###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2024 All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
###

set -e

for i in "cmd pkg utils"
#for i in "adapters"
do
    OUTPUT="$(goimports -l -e $i)"
    if [[ $OUTPUT ]]; then
        echo "The following files contain goimports errors"
        echo $OUTPUT
        echo "modify the OUTPUT file "
        for j in $OUTPUT
        do
            # modify the code format if the format is illegal
            echo $j
            goimports $j > "txt"
            cat "txt" > $j
        done
        echo "The goimports command must be run for these files"
        # exit 1
    fi
done
