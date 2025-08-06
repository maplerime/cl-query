#!/bin/sh
unset http_proxy
unset https_proxy

export GO111MODULE=on
export GOSUMDB=off
export GOPROXY=goproxy.io,direct
export GOPRIVATE=*raksmart.com

go mod tidy

