PACK             := defang-azure
PACKDIR          := sdk
PROJECT          := github.com/DefangLabs/pulumi-defang
NODE_MODULE_NAME := @defang-io/pulumi-defang-azure
NUGET_PKG_NAME   := DefangLabs.defang-azure

PROVIDER        := pulumi-resource-${PACK}
VERSION         ?= $(shell pulumictl get version $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease) | sed -E 's/\.([0-9]{10})(\+|$$)/\2/')
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}/defangazure.Version
IS_PRERELEASE   := $(shell git tag --sort=creatordate | tail -n1 | grep -q "alpha\|beta\|rc\|preview"; echo $$?)

GOPATH		:= $(shell go env GOPATH)

WORKING_DIR     := $(shell pwd)

OS    := $(shell uname)
SHELL := /bin/bash

.PHONY: provider
provider: $(WORKING_DIR)/bin/$(PROVIDER)
$(WORKING_DIR)/bin/$(PROVIDER): $(shell find . -name "*.go" -not -path "./sdk/*")
	go build -o "$(WORKING_DIR)/bin/${PROVIDER}" -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" "$(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER)"

.PHONY: schema
schema: provider
	pulumi package get-schema "$(WORKING_DIR)/bin/${PROVIDER}" > "${PROVIDER_PATH}/cmd/$(PROVIDER)/schema.json"

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: go_sdk
go_sdk: provider
	rm -rf "sdk/go/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language go -o "$(WORKING_DIR)/.sdk.tmp"
	mkdir -p sdk/go && mv "$(WORKING_DIR)/.sdk.tmp/go/defangazure" "sdk/go/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp"
	cd "sdk/go/${PACK}" && go mod init "$(PROJECT)/sdk/go/${PACK}" && \
		go get "github.com/pulumi/pulumi/sdk/v3@$(shell grep 'pulumi/pulumi/sdk/v3 ' $(WORKING_DIR)/go.mod | awk '{print $$2}')" && \
		go mod tidy

nodejs_sdk: VERSION := $(shell pulumictl get version --language javascript $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
.PHONY: nodejs_sdk
nodejs_sdk: provider
	rm -rf "sdk/nodejs/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language nodejs -o "$(WORKING_DIR)/.sdk.tmp"
	mkdir -p "sdk/nodejs/${PACK}" && cp -r "$(WORKING_DIR)/.sdk.tmp/nodejs/." "sdk/nodejs/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp"
	cd "${PACKDIR}/nodejs/${PACK}/" && \
		yarn install && \
		yarn run tsc && \
		sed -i.bak 's/$${VERSION}/$(VERSION)/g' package.json && \
		rm -f ./package.json.bak && \
		cp ../../../README.md ../../../LICENSE package.json yarn.lock bin/

python_sdk: PYPI_VERSION := $(shell pulumictl get version --language python $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
.PHONY: python_sdk
python_sdk: provider
	rm -rf "sdk/python/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language python -o "$(WORKING_DIR)/.sdk.tmp"
	mkdir -p "sdk/python/${PACK}" && cp -r "$(WORKING_DIR)/.sdk.tmp/python/." "sdk/python/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp"
	cp README.md "${PACKDIR}/python/${PACK}/"
	cd "${PACKDIR}/python/${PACK}/" && \
		python3 setup.py clean --all 2>/dev/null; \
		rm -rf ./bin/ ../python.bin/ && cp -R . ../python.bin && mv ../python.bin ./bin && \
		sed -i.bak -e 's/^VERSION = .*/VERSION = "$(PYPI_VERSION)"/g' -e 's/^PLUGIN_VERSION = .*/PLUGIN_VERSION = "$(VERSION)"/g' ./bin/setup.py && \
		rm -f ./bin/setup.py.bak && \
		cd ./bin && python3 setup.py build sdist

dotnet_sdk: DOTNET_VERSION := $(shell pulumictl get version --language dotnet $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
.PHONY: dotnet_sdk
dotnet_sdk: provider
	rm -rf "sdk/dotnet/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language dotnet -o "$(WORKING_DIR)/.sdk.tmp"
	mkdir -p "sdk/dotnet/${PACK}" && cp -r "$(WORKING_DIR)/.sdk.tmp/dotnet/." "sdk/dotnet/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp"
	cd "${PACKDIR}/dotnet/${PACK}/" && \
		echo "${DOTNET_VERSION}" >version.txt && \
		dotnet build /p:Version=${DOTNET_VERSION}

.PHONY: sdks
sdks: go_sdk nodejs_sdk python_sdk dotnet_sdk

.PHONY: build
build: provider schema sdks

.PHONY: install
install: provider
	cp "$(WORKING_DIR)/bin/${PROVIDER}" "${GOPATH}/bin"

.PHONY: clean
clean:
	rm -rf "$(WORKING_DIR)/bin/${PROVIDER}" "sdk/go/${PACK}" "sdk/nodejs/${PACK}" "sdk/python/${PACK}" "sdk/dotnet/${PACK}"
