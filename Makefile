# Naming
BINARY=spt-util
VET_REPORT=vet.report
# Docker
REGISTRY=registry.sptcloud.com
PROJECT=spt
IMAGE=${REGISTRY}/${PROJECT}/${BINARY}
# Go
GOARCH=$(shell go env GOARCH)
MODULE=$(shell go list -m)
GO_PKGS=$(foreach pkg, $(shell go list ./...), $(if $(findstring /vendor/, $(pkg)), , $(pkg)))
GO_FILES=$(shell find . -type f -name '*.go' -not -path './vendor/*')
LINT_TOOL=$(shell go env GOPATH)/bin/golangci-lint
# Build
VERSION=$(shell git describe --tags --always | sed 's/v//;s/-.*//')
REVISION=$(shell git rev-parse --short=7 HEAD)
PACKAGE=${MODULE}/pkg/version
OUTPUT_DIR=out
BUILD_OUTPUT=${OUTPUT_DIR}/bin/${BINARY}


all: help

## Build:
PLATFORMS := linux/${GOARCH} darwin/${GOARCH} windows/${GOARCH}
LDFLAGS = -ldflags "-X ${PACKAGE}.version=${VERSION} -X ${PACKAGE}.revision=${REVISION}"

temp=$(subst /, ,$@)
os=$(word 1, $(temp))
arch=$(word 2, $(temp))

release: $(PLATFORMS)  ## Build the project for target platforms.

$(PLATFORMS):
	env GOOS=${os} GOARCH=${arch} go build ${LDFLAGS} \
		-o ${BUILD_OUTPUT}-${os}-${arch} .

linux:  ## Build the project for Linux.
	env GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} \
		-o ${BUILD_OUTPUT}-linux-${GOARCH} .

darwin:  ## Build the project for MacOS.
	env GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} \
		-o ${BUILD_OUTPUT}-darwin-${GOARCH} .

windows:  ## Build the project for Windows.
	env GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} \
		-o ${BUILD_OUTPUT}-windows-${GOARCH}.exe .

clean:	## Remove build-related files.
	rm -rf ./${OUTPUT_DIR}/${TEST_REPORT}
	rm -rf ./${OUTPUT_DIR}/${VET_REPORT}
	rm -rf ./${OUTPUT_DIR}/bin

docker:	## Build the Docker container.
	docker build -f Dockerfile \
		-t ${IMAGE}:${VERSION} \
		-t ${IMAGE}:latest \
		--build-arg VERSION=${VERSION} \
		--build-arg REVISION=${REVISION} \
		--build-arg PACKAGE=${PACKAGE} .

## Test
test:  ## Run all of the project's tests.
	go test -v ./...

## Dependencies:
tidy:	## Run tidy and vendor to get the project's dependencies.
	go mod tidy
	go mod vendor

deps-reset:	## Reset the project's module dependencies.
	git checkout -- go.mod
	go mod tidy
	go mod vendor

deps-upgrade:	## Upgrade the project's dependencies.
	go get -u -t -d -v ./...

deps-cleancache:	## Clean the module cache.
	go clean -modcache

## Linting:
lint: lint-go ## Run the configured linting tools against the project.

$(LINT_TOOL):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.50.1

lint-go: $(LINT_TOOL)
	$(LINT_TOOL) run --config=.golangci.yaml ./...
	staticcheck ./...

fmt:  ## Run go formatting tools.
	@go fmt $(GO_PKGS)
	@goimports -w -l $(GO_FILES)

vet:  ## Run go vet tools.
	go vet ./... > ${OUTPUT_DIR}/${VET_REPORT} 2>&1

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

.PHONY: all release $(PLATFORMS) linux darwin windows clean $(TEST_RPT_TOOL) test docker \
	tidy deps-reset deps-upgrade deps-cleancache lint $(LINT_TOOL) lint-go fmt help
