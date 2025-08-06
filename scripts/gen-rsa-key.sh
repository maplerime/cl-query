#!/bin/bash
###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2019 -2020All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
#
#   Purpose:  generate rsa key
#
###

if [ -f "etc/private.pem" ]; then
    rm -f etc/private.pem
fi

if [ -f "etc/public.pem" ]; then
    rm -f etc/public.pem
fi

openssl genrsa -out etc/private.pem 4096
openssl rsa -in etc/private.pem -pubout -out etc/public.pem

echo "RSA key generated"
