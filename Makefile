## This is a self-documented Makefile. For usage information, run `make help`:
##
## For more information, refer to https://suva.sh/posts/well-documented-makefiles/

SHELL := /bin/bash

all: help

##@ Building
build: docker binary ##  Builds the application (same as 'docker')

set-version: ## Sets the version
	./ci/set-version.sh

docker: set-version ##  Builds the mheers/godtemplate application
	docker buildx build --platform linux/amd64 -t mheers/godtemplate --output type=docker .

docker-arm64: set-version ##  Builds the mheers/godtemplate application for arm64
	docker buildx build --platform linux/arm64 -t mheers/godtemplate --output type=docker .

docker-multi: set-version ##  Builds the mheers/godtemplate application for amd64 and arm64
	docker buildx build --platform linux/amd64,linux/arm64 -t mheers/godtemplate --push .

push:
	docker push mheers/godtemplate

binary: build-linux-amd64 build-windows-amd64 build-darwin-arm64

BINARY_NAME=godtemplate

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64


##@ Helpers

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
