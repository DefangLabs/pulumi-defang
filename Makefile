PROJECT_NAME := Pulumi defang Resource Provider

PACK             := defang
PACKDIR          := sdk
PROJECT          := github.com/DefangLabs/pulumi-defang
NODE_MODULE_NAME := @defang-io/pulumi-defang
NUGET_PKG_NAME   := DefangLabs.defang

PROVIDER        := pulumi-resource-${PACK}
VERSION         ?= $(shell pulumictl get version $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}.Version
IS_PRERELEASE   := $(shell git tag --sort=creatordate | tail -n1 | grep -q "alpha\|beta\|rc\|preview"; echo $$?)

GOPATH		:= $(shell go env GOPATH)

WORKING_DIR     := $(shell pwd)
EXAMPLES_DIR    := ${WORKING_DIR}/examples/yaml
TESTPARALLELISM := 4

OS    := $(shell uname)
SHELL := /bin/bash

EXAMPLE_STACK_NAME := example

prepare::
	@if test -z "${NAME}"; then echo "NAME not set"; exit 1; fi
	@if test -z "${REPOSITORY}"; then echo "REPOSITORY not set"; exit 1; fi
	@if test -z "${ORG}"; then echo "ORG not set"; exit 1; fi
	@if test ! -d "provider/cmd/pulumi-resource-defang"; then "Project already prepared"; exit 1; fi # SED_SKIP

	mv "provider/cmd/pulumi-resource-defang" provider/cmd/pulumi-resource-${NAME} # SED_SKIP

	if [[ "${OS}" != "Darwin" ]]; then \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '/SED_SKIP/!s,github.com/pulumi/pulumi-[x]yz,${REPOSITORY},g' {} \; &> /dev/null; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '/SED_SKIP/!s/[xX]yz/${NAME}/g' {} \; &> /dev/null; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '/SED_SKIP/!s/[aA]bc/${ORG}/g' {} \; &> /dev/null; \
	fi

	# In MacOS the -i parameter needs an empty string to execute in place.
	if [[ "${OS}" == "Darwin" ]]; then \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '' '/SED_SKIP/!s,github.com/pulumi/pulumi-[x]yz,${REPOSITORY},g' {} \; &> /dev/null; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '' '/SED_SKIP/!s/[xX]yz/${NAME}/g' {} \; &> /dev/null; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '' '/SED_SKIP/!s/[aA]bc/${ORG}/g' {} \; &> /dev/null; \
	fi

.PHONY: ensure
ensure:
	cd provider && go mod tidy
	cd sdk && go mod tidy
	cd tests && go mod tidy

provider: $(WORKING_DIR)/bin/$(PROVIDER)
$(WORKING_DIR)/bin/$(PROVIDER): $(shell find . -name "*.go")
	go build -o $(WORKING_DIR)/bin/${PROVIDER} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER)

.PHONY: provider_debug
provider_debug:
	(cd provider && go build -o $(WORKING_DIR)/bin/${PROVIDER} -gcflags="all=-N -l" -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER))

.PHONY: schema
schema: provider
	pulumi package get-schema $(WORKING_DIR)/bin/${PROVIDER} > ${PROVIDER_PATH}/cmd/$(PROVIDER)/schema.json

.PHONY: test_provider
test_provider:
	cd tests && go test -short -v -count=1 -cover -timeout 5m -parallel ${TESTPARALLELISM} ./...

.PHONY: version
version:
	@echo $(VERSION)

dotnet_sdk: DOTNET_VERSION := $(shell pulumictl get version --language dotnet $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
dotnet_sdk: provider
	rm -rf sdk/dotnet
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language dotnet
	cd ${PACKDIR}/dotnet/&& \
		echo "${DOTNET_VERSION}" >version.txt && \
		dotnet build /p:Version=${DOTNET_VERSION}

.PHONY: go_sdk
go_sdk: provider
	rm -rf sdk/go
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language go

.PHONY: nodejs_sdk
nodejs_sdk: VERSION := $(shell pulumictl get version --language javascript $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
nodejs_sdk: provider
	rm -rf sdk/nodejs
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language nodejs
	cd ${PACKDIR}/nodejs/ && \
		yarn install && \
		yarn run tsc && \
		sed -i.bak 's/$${VERSION}/$(VERSION)/g' package.json && \
		rm ./package.json.bak && \
		cp ../../README.md ../../LICENSE package.json yarn.lock bin/

.PHONY: python_sdk
python_sdk: PYPI_VERSION := $(shell pulumictl get version --language python $(if $(filter 0,$(IS_PRERELEASE)),--is-prerelease))
python_sdk: provider
	rm -rf sdk/python
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language python
	cp README.md ${PACKDIR}/python/
	cd ${PACKDIR}/python/ && \
		python3 setup.py clean --all 2>/dev/null && \
		rm -rf ./bin/ ../python.bin/ && cp -R . ../python.bin && mv ../python.bin ./bin && \
		sed -i.bak -e 's/^VERSION = .*/VERSION = "$(PYPI_VERSION)"/g' -e 's/^PLUGIN_VERSION = .*/PLUGIN_VERSION = "$(VERSION)"/g' ./bin/setup.py && \
		rm ./bin/setup.py.bak && \
		cd ./bin && python3 setup.py build sdist

.PHONY: examples
examples: go_example \
		nodejs_example \
		python_example \
		dotnet_example

%_example:
	rm -rf ${WORKING_DIR}/examples/$*
	pulumi convert \
		--cwd ${WORKING_DIR}/examples/yaml \
		--logtostderr \
		--generate-only \
		--non-interactive \
		--language $* \
		--out ${WORKING_DIR}/examples/$*

.PHONY: docs
docs: README.md
	cp docs/.front-matter.md docs/_index.md
	cat README.md >> docs/_index.md

define pulumi_login
    export PULUMI_CONFIG_PASSPHRASE=asdfqwerty1234; \
    pulumi login --local;
endef

up::
	$(call pulumi_login) \
	cd ${EXAMPLES_DIR} && \
	(pulumi stack select ${EXAMPLE_STACK_NAME} || pulumi stack init ${EXAMPLE_STACK_NAME}) && \
	pulumi config set name ${EXAMPLE_STACK_NAME} && \
	pulumi up -y

down::
	$(call pulumi_login) \
	cd ${EXAMPLES_DIR} && \
	pulumi stack select ${EXAMPLE_STACK_NAME} && \
	pulumi destroy -y && \
	pulumi stack rm ${EXAMPLE_STACK_NAME} -y

.PHONY: build
build: provider schema sdks

.PHONY: sdks
sdks: go_sdk nodejs_sdk python_sdk dotnet_sdk

.PHONY: only_build
# Required for the codegen action that runs in pulumi/pulumi
only_build: build

.PHONY: lint
lint:
	golangci-lint run --fix --timeout 5m ./provider ./tests

.PHONY: install
install: install_nodejs_sdk install_dotnet_sdk
	cp $(WORKING_DIR)/bin/${PROVIDER} ${GOPATH}/bin

GO_TEST	 := go test -v -count=1 -cover -timeout 5m -parallel ${TESTPARALLELISM}

.PHONY: test
test: test_provider

.PHONY: test_all
test_all: test
	cd tests/sdk/nodejs && $(GO_TEST) ./...
	cd tests/sdk/python && $(GO_TEST) ./...
	cd tests/sdk/go && $(GO_TEST) ./...
	# cd tests/sdk/dotnet && $(GO_TEST) ./...

install_dotnet_sdk:
	rm -rf $(WORKING_DIR)/nuget/$(NUGET_PKG_NAME).*.nupkg
	mkdir -p $(WORKING_DIR)/nuget
	find . -name '*.nupkg' -print -exec cp -p {} ${WORKING_DIR}/nuget \;

.PHONY: install_python_sdk
install_python_sdk:
	#target intentionally blank

.PHONY: install_go_sdk
install_go_sdk:
	#target intentionally blank

.PHONY: install_nodejs_sdk
install_nodejs_sdk:
	-yarn unlink --cwd $(WORKING_DIR)/sdk/nodejs/bin
	yarn link --cwd $(WORKING_DIR)/sdk/nodejs/bin

.PHONY: install-git-hooks
install-git-hooks:
	printf "#!/bin/sh\nmake pre-commit" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	printf "#!/bin/sh\nmake pre-push" > .git/hooks/pre-push
	chmod +x .git/hooks/pre-push

.PHONY: pre-commit
pre-commit: provider test lint examples docs
	git add examples

.PHONY: pre-push
pre-push:
	#target intentionally blank
