SHELL=/bin/bash -o pipefail
PROJECT_NAME := Pulumi defang Resource Provider

PACKS           := defang-aws defang-gcp defang-azure
PROJECT         := github.com/DefangLabs/pulumi-defang
PROVIDER_PATH   := provider

.PHONY: help
help: ## Show this help message
	@grep -hE '^[a-zA-Z0-9_%-]+:.*##' $(MAKEFILE_LIST) \
		| sort \
		| awk -F':|##' '{printf "  \033[36m%-22s\033[0m %s\n", $$1, $$3}'

GOPATH		:= $(shell go env GOPATH)
export GOTOOLCHAIN := go1.25.6

WORKING_DIR     := $(shell pwd)
TESTPARALLELISM := 4

OS    := $(shell uname)

# Delegate to per-plugin Makefiles
define pack_rule
%_$(1):
	$$(MAKE) -f $(1).mk $$*
endef
$(foreach p,$(PACKS),$(eval $(call pack_rule,$(p))))

# Aggregate targets
.PHONY: provider
provider: $(foreach p,$(PACKS),provider_$(p)) ## Build all provider binaries

.PHONY: schema
schema: $(foreach p,$(PACKS),schema_$(p)) ## Generate OpenAPI schemas

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
sdks: go_sdk nodejs_sdk python_sdk dotnet_sdk ## Generate all language SDKs

.PHONY: build
build: provider schema sdks ## Full build: provider + schema + sdks

.PHONY: only_build
# Required for the codegen action that runs in pulumi/pulumi
only_build: build

.PHONY: ensure
ensure: ## Run go mod tidy
	go mod tidy

GO_TEST	 := go test -v -count=1 -cover -timeout 5m -parallel ${TESTPARALLELISM}

.PHONY: test_provider
test_provider: provider ## Provider integration tests
	cd tests && ${GO_TEST} -coverprofile=../coverage_tests.out -coverpkg=github.com/DefangLabs/pulumi-defang/provider/... -short ./... | sed -e 's/\(--- FAIL.*\)/[0;31m\1[0m/g'

.PHONY: test_unit
test_unit: ## Unit tests only
	${GO_TEST} -coverprofile=coverage_provider.out ./provider/... | sed -e 's/\(--- FAIL.*\)/[0;31m\1[0m/g'

.PHONY: test
test: test_unit test_provider ## Run all tests

.PHONY: coverage
coverage: test
	cat coverage_provider.out <(tail -n +2 coverage_tests.out) > coverage.out
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

.PHONY: version
version: ## Print the current version based on Git tags
	@$(MAKE) --no-print-directory -f defang-aws.mk version

.PHONY: lint
lint: ## Run linter with --fix
	golangci-lint run --fix --timeout 5m ./provider/... ./tests/...

.PHONY: install
install: $(foreach p,$(PACKS),install_$(p)) ## Install providers to $GOPATH/bin

README_SOURCES := examples/aws-nodejs/index.ts examples/aws-python/__main__.py examples/aws-go/main.go examples/aws-dotnet/Program.cs examples/aws-yaml/Pulumi.yaml
README.md: $(README_SOURCES) scripts/check-readme-examples.sh ## Regenerate README.md code blocks from examples/
	scripts/check-readme-examples.sh --update

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
install-git-hooks: node_modules ## Set up pre-commit and pre-push hooks
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
examples: $(foreach p,$(EXAMPLE_PROVIDERS),gen_examples_$(p)) ## Generate language examples from YAML
	$(MAKE) README.md

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
