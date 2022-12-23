MODULE = $(shell go list -m)
SHELL := /bin/bash
LINT_TOOL=$(shell go env GOPATH)/bin/golangci-lint
GO_PKGS=$(foreach pkg, $(shell go list ./...), $(if $(findstring /vendor/, $(pkg)), , $(pkg)))
GO_FILES=$(shell find . -type f -name '*.go' -not -path './vendor/*')

VERSION=$(shell git describe --tags --always | sed 's/v//;s/-.*//')
REVISION=$(shell git rev-parse --short=7 HEAD)
PACKAGE="github.com/robertwtucker/spt-util/internal/config"
BINARY=spt-util
IMAGE="registry.sptcloud.com/spt/${BINARY}"

OUTPUT_DIR=out/bin/${BINARY}

ENV := local
ifdef $$APP_ENV
ENV := $$APP_ENV
endif

export PROJECT = github.com/robertwtucker/spt-util

.PHONY: all test build vendor

all: help

## Build:
build:	## Build the project for Linux.
	env GOOS=linux GOARCH=amd64 go build \
		-ldflags "-X ${PACKAGE}.appVersion=${VERSION} -X ${PACKAGE}.revision=${REVISION}" \
		-o ${OUTPUT_DIR} ./main.go
	chmod +x ${OUTPUT_DIR}

build-mac:	## Build the project for MacOS.
	env GOOS=darwin GOARCH=amd64 go build \
		-ldflags "-X ${PACKAGE}.appVersion=${VERSION} -X ${PACKAGE}.revision=${REVISION}" \
		-o ${OUTPUT_DIR} ./main.go
	chmod +x ${OUTPUT_DIR}

build-win:	## Build the project for Windows.
	env GOOS=windows GOARCH=amd64 go build \
		-ldflags "-X ${PACKAGE}.appVersion=${VERSION} -X ${PACKAGE}.revision=${REVISION}" \
		-o ${OUTPUT_DIR} ./main.go

clean:	## Remove build-related files.
	rm -rf ./bin
	rm -rf ./out

## Run:
run:	## Run the project from source.
	go run ./main.go

start:	## Run the project binary.
	./${OUTPUT_DIR}

## Test
test:	## Run all of the project's tests.
	go test ./... -count=1

## Docker:
docker:	## Build the container for Docker.
	docker build -t ${IMAGE}:latest -t ${IMAGE}:${VERSION} \
		--build-arg BUILD_VERSION=${VERSION} --build-arg BUILD_REVISION=${REVISION} .

## Dependencies:
tidy:	## Run tidy and vendor to get the project's dependencies.
	go mod tidy -compat=1.17
	go mod vendor

deps-reset:	## Reset the project's module dependencies.
	git checkout -- go.mod
	go mod tidy
	go mod vendor

deps-upgrade:	## Upgrade the project's dependencies.
	go get -u -t -d -v ./...

deps-cleancache:	## Clean the module cache.
	go clean -modcache

fmt:
	@go fmt $(GO_PKGS)
	@goimports -w -l $(GO_FILES)

## Lint:
lint: lint-go

$(LINT_TOOL):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.50.1

lint-go: $(LINT_TOOL) ## Run golangci-lint against the project.
	$(LINT_TOOL) run --config=.golangci.yaml ./...
	staticcheck ./...

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

## Help:
help:	## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
