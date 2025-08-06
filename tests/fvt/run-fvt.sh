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

virtualenv -p python3 venv
source venv/bin/activate

pip install -r tests/fvt/requirements.txt

pytest -v tests/fvt/*.py