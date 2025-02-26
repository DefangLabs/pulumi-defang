PROJECT_NAME := Pulumi defang Resource Provider

PACK             := defang
PACKDIR          := sdk
PROJECT          := github.com/DefangLabs/pulumi-defang
NODE_MODULE_NAME := @DefangLabs/defang
NUGET_PKG_NAME   := DefangLabs.defang

PROVIDER        := pulumi-resource-${PACK}
VERSION         ?= $(shell pulumictl get version)
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}.Version

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

.PHONY: provider
provider: $(WORKING_DIR)/bin/$(PROVIDER)
$(WORKING_DIR)/bin/$(PROVIDER): $(shell find . -name "*.go")
	go build -o $(WORKING_DIR)/bin/${PROVIDER} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER)

.PHONY: provider_debug
provider_debug:
	(cd provider && go build -o $(WORKING_DIR)/bin/${PROVIDER} -gcflags="all=-N -l" -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER))

.PHONY: test_provider
test_provider:
	cd tests && go test -short -v -count=1 -cover -timeout 2h -parallel ${TESTPARALLELISM} ./...

# dotnet_sdk: DOTNET_VERSION := $(shell pulumictl get version --language dotnet)
# dotnet_sdk: $(WORKING_DIR)/bin/$(PROVIDER)
# 	rm -rf sdk/dotnet
# 	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language dotnet
# 	cd ${PACKDIR}/dotnet/&& \
# 		echo "${DOTNET_VERSION}" >version.txt && \
# 		dotnet build /p:Version=${DOTNET_VERSION}

.PHONY: go_sdk
go_sdk: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf sdk/go
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language go

.PHONY: nodejs_sdk
nodejs_sdk: VERSION := $(shell pulumictl get version --language javascript)
nodejs_sdk: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf sdk/nodejs
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language nodejs
	cd ${PACKDIR}/nodejs/ && \
		yarn install && \
		yarn run tsc && \
		cp ../../README.md ../../LICENSE package.json yarn.lock bin/ && \
		sed -i.bak 's/$${VERSION}/$(VERSION)/g' bin/package.json && \
		rm ./bin/package.json.bak

.PHONY: python_sdk
python_sdk: PYPI_VERSION := $(shell pulumictl get version --language python)
python_sdk: $(WORKING_DIR)/bin/$(PROVIDER)
	rm -rf sdk/python
	pulumi package gen-sdk $(WORKING_DIR)/bin/$(PROVIDER) --language python
	cp README.md ${PACKDIR}/python/
	cd ${PACKDIR}/python/ && \
		python3 setup.py clean --all 2>/dev/null && \
		rm -rf ./bin/ ../python.bin/ && cp -R . ../python.bin && mv ../python.bin ./bin && \
		sed -i.bak -e 's/^VERSION = .*/VERSION = "$(PYPI_VERSION)"/g' -e 's/^PLUGIN_VERSION = .*/PLUGIN_VERSION = "$(VERSION)"/g' ./bin/setup.py && \
		rm ./bin/setup.py.bak && \
		cd ./bin && python3 setup.py build sdist

.PHONY: gen_examples
gen_examples: gen_go_example \
		gen_nodejs_example \
		gen_python_example \
		# gen_dotnet_example

.PHONY: gen_go_example
.PHONY: gen_nodejs_example
.PHONY: gen_python_example
gen_%_example:
	rm -rf ${WORKING_DIR}/examples/$*
	pulumi convert \
		--cwd ${WORKING_DIR}/examples/yaml \
		--logtostderr \
		--generate-only \
		--non-interactive \
		--language $* \
		--out ${WORKING_DIR}/examples/$*

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
# build: provider dotnet_sdk go_sdk nodejs_sdk python_sdk
build: provider go_sdk nodejs_sdk python_sdk

.PHONY: only_build
# Required for the codegen action that runs in pulumi/pulumi
only_build: build

.PHONY: lint
lint:
	for DIR in "provider" "tests" ; do \
		pushd $$DIR && golangci-lint run --fix --timeout 10m && popd ; \
	done

.PHONY: install
# install: install_nodejs_sdk install_dotnet_sdk
# 	cp $(WORKING_DIR)/bin/${PROVIDER} ${GOPATH}/bin
install: install_nodejs_sdk
	cp $(WORKING_DIR)/bin/${PROVIDER} ${GOPATH}/bin

GO_TEST	 := go test -v -count=1 -cover -timeout 2h -parallel ${TESTPARALLELISM}

.PHONY: test
test: test_provider

.PHONY: test_all
test_all: test
	cd tests/sdk/nodejs && $(GO_TEST) ./...
	cd tests/sdk/python && $(GO_TEST) ./...
	cd tests/sdk/go && $(GO_TEST) ./...
	# cd tests/sdk/dotnet && $(GO_TEST) ./...

# install_dotnet_sdk:
# 	rm -rf $(WORKING_DIR)/nuget/$(NUGET_PKG_NAME).*.nupkg
# 	mkdir -p $(WORKING_DIR)/nuget
# 	find . -name '*.nupkg' -print -exec cp -p {} ${WORKING_DIR}/nuget \;

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
