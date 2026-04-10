SHELL=/bin/bash -o pipefail
PROJECT_NAME := Pulumi defang Resource Provider

PACKS           := defang-aws defang-gcp defang-azure
PROJECT         := github.com/DefangLabs/pulumi-defang
PROVIDER_PATH   := provider

GOPATH		:= $(shell go env GOPATH)
export GOTOOLCHAIN := go1.25.6

WORKING_DIR     := $(shell pwd)
TESTPARALLELISM := 4

OS    := $(shell uname)
SHELL := /bin/bash

# Delegate to per-plugin Makefiles
define plugin_targets
.PHONY: provider_$(1) schema_$(1) go_sdk_$(1) nodejs_sdk_$(1) python_sdk_$(1) dotnet_sdk_$(1) sdks_$(1) build_$(1) clean_$(1) install_$(1)
provider_$(1) schema_$(1) go_sdk_$(1) nodejs_sdk_$(1) python_sdk_$(1) dotnet_sdk_$(1) sdks_$(1) build_$(1) clean_$(1) install_$(1):
	$$(MAKE) -f $(1).mk $$(patsubst %_$(1),%,$$@)
endef

$(foreach p,$(PACKS),$(eval $(call plugin_targets,$(p))))

# Aggregate targets
.PHONY: provider
provider: $(foreach p,$(PACKS),provider_$(p))

.PHONY: schema
schema: $(foreach p,$(PACKS),schema_$(p))

.PHONY: go_sdk
go_sdk: $(foreach p,$(PACKS),go_sdk_$(p))
	@! egrep -r '^type .+Input\w*Input' sdk/v2 || (echo "Error: InputInput types found in Go SDK code" ; false)
	@! egrep -r '^type .+Output\w*Output' sdk/v2 || (echo "Error: OutputOutput types found in Go SDK code" ; false)

.PHONY: nodejs_sdk
nodejs_sdk: $(foreach p,$(PACKS),nodejs_sdk_$(p))

.PHONY: python_sdk
python_sdk: $(foreach p,$(PACKS),python_sdk_$(p))

.PHONY: dotnet_sdk
dotnet_sdk: $(foreach p,$(PACKS),dotnet_sdk_$(p))

.PHONY: sdks
sdks: go_sdk nodejs_sdk python_sdk dotnet_sdk

.PHONY: build
build: provider schema sdks

.PHONY: only_build
# Required for the codegen action that runs in pulumi/pulumi
only_build: build

.PHONY: ensure
ensure:
	go mod tidy

GO_TEST	 := go test -v -count=1 -cover -timeout 5m -parallel ${TESTPARALLELISM}

.PHONY: test_provider
test_provider: provider
	cd tests && ${GO_TEST} -coverprofile=../coverage_tests.out -coverpkg=github.com/DefangLabs/pulumi-defang/provider/... -short ./... | sed -e 's/\(--- FAIL.*\)/[0;31m\1[0m/g'

.PHONY: test_unit
test_unit:
	${GO_TEST} -coverprofile=coverage_provider.out ./provider/... | sed -e 's/\(--- FAIL.*\)/[0;31m\1[0m/g'

.PHONY: test
test: test_unit test_provider

.PHONY: coverage
coverage: test
	cat coverage_provider.out <(tail -n +2 coverage_tests.out) > coverage.out
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

.PHONY: version
version:
	@$(MAKE) -f defang-aws.mk version

.PHONY: lint
lint:
	golangci-lint run --fix --timeout 5m ./provider/... ./tests/...

.PHONY: install
install: $(foreach p,$(PACKS),install_$(p))

.PHONY: clean
clean: $(foreach p,$(PACKS),clean_$(p))

.PHONY: release
release: clean build

# Docker images
DOCKER_BUILDX  := docker buildx build
IMAGE_REPO     := defangio/cd
CD_VERSION     := $(shell git describe --tags --always --dirty)
PROVIDER_VERSION := $(shell $(MAKE) -s -f defang-aws.mk version)

.PHONY: images
images: image_aws image_gcp image_azure image_all

image_%:
	$(DOCKER_BUILDX) --build-arg CLOUDS=$* \
	  --build-arg CD_VERSION=$(CD_VERSION) --build-arg PROVIDER_VERSION=$(PROVIDER_VERSION) \
	  -t $(IMAGE_REPO):$(CD_VERSION)-$* .

.PHONY: install-git-hooks
install-git-hooks: node_modules
	printf "#!/bin/sh\nmake pre-commit" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	printf "#!/bin/sh\nmake -j4 pre-push" > .git/hooks/pre-push
	chmod +x .git/hooks/pre-push

node_modules: package.json
	npm install

# Fast pre-commit: lint-staged runs golangci-lint + go test only for changed providers.
# Shared packages (provider/compose, provider/common) trigger all providers.
.PHONY: pre-commit
pre-commit: node_modules
	npx --no lint-staged

# Full build + test run before push (or for CI).
.PHONY: pre-push
pre-push: provider test go_sdk

# Generate language examples from YAML sources
# Requires providers to be built first: make install
EXAMPLE_PROVIDERS  := aws gcp azure
EXAMPLE_LANGUAGES  := go nodejs python dotnet

.PHONY: examples
examples: $(foreach p,$(EXAMPLE_PROVIDERS),gen_examples_$(p))

define example_target
.PHONY: example_$(1)_$(2)
example_$(1)_$(2): install_defang-$(1)
	cd examples/$(1)-yaml && pulumi convert --language $(2) --generate-only --out ../$(1)-$(2)
endef

$(foreach p,$(EXAMPLE_PROVIDERS),$(foreach l,$(EXAMPLE_LANGUAGES),$(eval $(call example_target,$(p),$(l)))))

define gen_examples_provider_target
.PHONY: gen_examples_$(1)
gen_examples_$(1): $(foreach l,$(EXAMPLE_LANGUAGES),example_$(1)_$(l))
endef

$(foreach p,$(EXAMPLE_PROVIDERS),$(eval $(call gen_examples_provider_target,$(p))))
