#!/bin/sh

export GOROOT=/usr/local/go
export GOPATH=/data/workspace/go
export PATH=$PATH:${GOROOT}/bin:${GOPATH}/bin

make checks