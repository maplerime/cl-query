#!/bin/bash
###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2019 -2020All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
#
#   Purpose:  generate ed25519 key
#
###

if [ -f "etc/api_key_private.pem" ]; then
    rm -f etc/api_key_private.pem
fi

if [ -f "etc/api_key_public.pem" ]; then
    rm -f etc/api_key_public.pem
fi

openssl genpkey -algorithm ed25519 -out etc/api_key_private.pem
openssl pkey -in etc/api_key_private.pem -pubout -out etc/api_key_public.pem

echo "ED25519 key generated"
