###
 # Licensed Materials - Property of PEG TECH INC
 #
 # (C) Copyright PEG TECH INC. 2024 All Rights Reserved
 #
 # Contributors:
 #    bryan@raksmart.com - Initial implementation
###

#
# -------------------------------------------------------------
# This makefile defines the following targets
#
#   - all (default) - builds all targets and runs all tests/checks
#   - setup - sets up the go module
#   - checks - runs all tests/checks
#   - unit-test - runs the go-test based unit tests
#   - linter - runs all code checks
#   - docs - generates the swagger docs
#   - clean - cleans up the build artifacts
#   - image - builds the docker image

ROOT_DIR ?= $(shell git rev-parse --show-toplevel)
VERSION ?= 1.0.0

# Color
no_color = \033[0m
red = \033[0;31m
green = \033[0;32m
yellow = \033[0;33m
blue = \033[0;34m
purple = \033[0;35m
cyan = \033[0;36m
white = \033[0;37m

# Version
# RELEASE_VERSION ?= $(shell git rev-parse --short HEAD)_$(shell date -u +%Y-%m-%dT%H:%M:%S%z)
RELEASE_VERSION ?= $(VERSION)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT ?= $(shell git rev-parse --verify HEAD)

PROJECT_NAME=cl-query
PKGNAME = github.com/maplerime/$(PROJECT_NAME)

IS_RELEASE = false

ARCH=$(shell uname -m)

# docker build
BUILD_ENGINE = docker
BUILD_CONTEXT = .
DOCKER_FILE = Dockerfile
IMAGE = $(PROJECT_NAME)
IMAGE_TAG = $(VERSION)-$(shell date -u +%Y%m%d%H%M%S)

all: checks build

setup:
#	rm -f go.mod go.sum
#	GO111MODULE=on GOSUMDB=off GOPRIVATE=*raksmart.com go mod init $(PKGNAME)
	rm -f go.sum
	go mod tidy
	go get golang.org/x/lint
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/onsi/gomega
	go get golang.org/x/tools/cmd/goimports
	go get github.com/microcosm-cc/bluemonday@v1.0.21
	go get github.com/gin-gonic/gin/binding@v1.10.0
	go get github.com/mattn/go-isatty@v0.0.20
	go get github.com/gin-gonic/gin@v1.10.0
	go get github.com/swaggo/files@v1.0.1
	go get github.com/PuerkitoBio/purell@v1.1.1
	go get github.com/spf13/afero@v1.15.0
	go get github.com/felixge/fgprof@v0.9.3
	go install golang.org/x/tools/cmd/goimports@latest

docs-setup: setup
	go get github.com/swaggo/swag/cmd/swag
	go install github.com/swaggo/swag/cmd/swag@latest
	go get github.com/swaggo/gin-swagger
	go get github.com/swaggo/files

error_codes_string:
	go generate pkg/common/error_codes.go

query-docs: docs-setup
	swag init -g router.go -dir ./cmd/querysvc,./pkg/api/resources,./pkg/common -o ./docs/

docs: query-docs

checks: docs linter unit-test

unit-test: setup
	@./scripts/goUnitTests.sh

linter: setup
	@echo "LINT: Running code checks.."
	@./scripts/verify-golangci-lint.sh
	@./scripts/goimports.sh

svcs: docs querysvc

.PHONY: querysvc
querysvc: build/bin/querysvc

build/bin/%: setup
	@mkdir -p $(@D)
	@echo "$@"
	$(CGO_FLAGS) GOBIN=$(abspath $(@D)) go install $(EXT_LDFLAGS) $(PKGNAME)/cmd/$(@F)
	@echo "Binary available as $@"
	@touch $@

.PHONY: clean
clean:
	rm -f build/bin/*

.PHONY: build
ifeq ($(BUILD_ENGINE), docker)
    build_cmd = docker build
else ifeq ($(BUILD_ENGINE), buildah)
    build_cmd = buildah bud
else
    $(error Unsupported build engine $(BUILD_ENGINE))
endif
build: svcs
	echo "$(IMAGE_TAG)" > version
	$(build_cmd) --network host --no-cache --force-rm --build-arg RELEASE_VERSION=$(RELEASE_VERSION)  --build-arg GIT_BRANCH=$(GIT_BRANCH) --build-arg GIT_COMMIT=$(GIT_COMMIT) $(BUILD_ARGS) -f $(DOCKER_FILE) -t $(IMAGE):$(IMAGE_TAG) $(BUILD_CONTEXT)
	echo "$(IMAGE)" > image_name
	echo "$(IMAGE):$(IMAGE_TAG)" > image.latest
