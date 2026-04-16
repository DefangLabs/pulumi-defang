PACK             := defang-gcp
PACKDIR          := sdk
PROJECT          := github.com/DefangLabs/pulumi-defang
NODE_MODULE_NAME := @defang-io/pulumi-defang-gcp
NUGET_PKG_NAME   := DefangLabs.defang-gcp

PROVIDER        := pulumi-resource-${PACK}
LATEST_TAG      := $(shell git tag --sort=creatordate | tail -n1)
VERSION_PREFIX  := $(shell echo $(LATEST_TAG) | sed -E 's/v?([^-]+).*/\1/')
IS_PRERELEASE   := $(shell echo $(LATEST_TAG) | grep -q "alpha\|beta\|rc\|preview"; echo $$?)
VERSION         ?= $(shell pulumictl get version --version-prefix $(VERSION_PREFIX) $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease) | sed -E 's/\.([0-9]{10})(\+|$$)/\2/')
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}/defanggcp.Version

GOPATH		:= $(shell go env GOPATH)

WORKING_DIR     := $(shell pwd)

SDK_GO_DIR       := sdk/v2/go/$(PACK)
SDK_MODULE       := $(PROJECT)/sdk/v2/go/$(PACK)

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
go_sdk: .sdk.go.$(PACK).stamp

.sdk.go.$(PACK).stamp: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf "$(SDK_GO_DIR)"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language go -o "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	mkdir -p "$(dir $(SDK_GO_DIR))" && mv "$(WORKING_DIR)/.sdk.tmp.$(PACK)/go/defanggcp" "$(SDK_GO_DIR)" && rm -rf "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	cd "$(SDK_GO_DIR)" && go mod init "$(SDK_MODULE)" && \
		go get "github.com/pulumi/pulumi/sdk/v3@$(shell grep 'pulumi/pulumi/sdk/v3 ' $(WORKING_DIR)/go.mod | awk '{print $$2}')" && \
		go mod tidy
	@touch $@

nodejs_sdk: VERSION := $(shell pulumictl get version --language javascript --version-prefix $(VERSION_PREFIX) $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease) | sed 's/^v//')
.PHONY: nodejs_sdk
nodejs_sdk: .sdk.nodejs.$(PACK).stamp

.sdk.nodejs.$(PACK).stamp: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf "sdk/nodejs/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language nodejs -o "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	mkdir -p "sdk/nodejs/${PACK}" && cp -r "$(WORKING_DIR)/.sdk.tmp.$(PACK)/nodejs/." "sdk/nodejs/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	cd "${PACKDIR}/nodejs/${PACK}/" && \
		yarn install && \
		yarn run tsc && \
		sed -i.bak 's/$${VERSION}/$(VERSION)/g' package.json && \
		rm -f ./package.json.bak && \
		cp ../../../README.md ../../../LICENSE package.json yarn.lock bin/
	@touch $@

python_sdk: PYPI_VERSION := $(shell pulumictl get version --language python --version-prefix $(VERSION_PREFIX) $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
.PHONY: python_sdk
python_sdk: .sdk.python.$(PACK).stamp

.sdk.python.$(PACK).stamp: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf "sdk/python/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language python -o "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	mkdir -p "sdk/python/${PACK}" && cp -r "$(WORKING_DIR)/.sdk.tmp.$(PACK)/python/." "sdk/python/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	cp README.md "${PACKDIR}/python/${PACK}/"
	cd "${PACKDIR}/python/${PACK}/" && \
		python3 setup.py clean --all 2>/dev/null; \
		rm -rf ./bin/ ../python.bin.$(PACK)/ && cp -R . ../python.bin.$(PACK) && mv ../python.bin.$(PACK) ./bin && \
		sed -i.bak -e 's/^VERSION = .*/VERSION = "$(PYPI_VERSION)"/g' -e 's/^PLUGIN_VERSION = .*/PLUGIN_VERSION = "$(VERSION)"/g' ./bin/setup.py && \
		rm -f ./bin/setup.py.bak && \
		cd ./bin && python3 setup.py build sdist
	@touch $@

dotnet_sdk: DOTNET_VERSION := $(shell pulumictl get version --language dotnet --version-prefix $(VERSION_PREFIX) $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
.PHONY: dotnet_sdk
dotnet_sdk: .sdk.dotnet.$(PACK).stamp

.sdk.dotnet.$(PACK).stamp: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf "sdk/dotnet/${PACK}"
	pulumi package gen-sdk "$(WORKING_DIR)/bin/$(PROVIDER)" --language dotnet -o "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	mkdir -p "sdk/dotnet/${PACK}" && cp -r "$(WORKING_DIR)/.sdk.tmp.$(PACK)/dotnet/." "sdk/dotnet/${PACK}" && rm -rf "$(WORKING_DIR)/.sdk.tmp.$(PACK)"
	cd "${PACKDIR}/dotnet/${PACK}/" && \
		echo "${DOTNET_VERSION}" >version.txt && \
		dotnet build /p:Version=${DOTNET_VERSION}
	@touch $@

.PHONY: sdks
sdks: go_sdk nodejs_sdk python_sdk dotnet_sdk

.PHONY: build
build: provider schema sdks

.PHONY: install
install: provider
	cp "$(WORKING_DIR)/bin/${PROVIDER}" "${GOPATH}/bin"

.PHONY: clean
clean:
	rm -rf "$(WORKING_DIR)/bin/${PROVIDER}" "$(SDK_GO_DIR)" "sdk/nodejs/${PACK}" "sdk/python/${PACK}" "sdk/dotnet/${PACK}"
	rm -f .sdk.go.$(PACK).stamp .sdk.nodejs.$(PACK).stamp .sdk.python.$(PACK).stamp .sdk.dotnet.$(PACK).stamp
