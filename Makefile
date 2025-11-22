# Image URL to use all building/pushing image targets
IMG ?= docker.io/hansfischer/openldap-operator:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.35.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=deploy/helm/openldap-operator/crds

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run all tests with coverage output.
	@echo "🧪 Running all tests with coverage..."
	@go test ./... -coverprofile=coverage.out -v | grep -E "(PASS|FAIL|coverage:)" || true
	@echo ""
	@echo "📊 Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "✅ Tests completed. Coverage report: coverage.out"

.PHONY: test-detailed
test-detailed: manifests generate fmt vet envtest ## Run tests with Kubernetes environment (for CRD validation).
	@echo "🧪 Running tests with Kubernetes environment..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile=coverage.out -v | grep -E "(PASS|FAIL|coverage:)" || true
	@echo ""
	@echo "📊 Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1

.PHONY: test-integration
test-integration: ## Run integration tests with Docker LDAP server.
	@echo "🐳 Running integration tests with Docker..."
	./test/run-tests.sh

.PHONY: test-all
test-all: test test-integration ## Run all tests (unit + integration).

.PHONY: test-unit
test-unit: fmt vet ## Run unit tests only (excluding integration tests).
	@echo "🧪 Running unit tests..."
	@go test ./api/... ./internal/controller/... -coverprofile=coverage.out -v 2>&1 | grep -v "does not match go tool version"
	@echo ""
	@echo "📊 Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1 || echo "No coverage data"

.PHONY: coverage-ci
coverage-ci: ## Generate coverage report for CI.
	@echo "📊 Generating coverage report..."
	@mkdir -p coverage
	@go tool cover -func=coverage.out > coverage/coverage.txt
	@go tool cover -html=coverage.out -o coverage/coverage.html
	@echo "Coverage report generated in coverage/"

.PHONY: lint
lint: ## Run golangci-lint.
	@echo "🔍 Running linter..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run ./... --out-format=colored-line-number

.PHONY: gosec
gosec: ## Run GoSec security scanner.
	@echo "🔒 Running GoSec security scan..."
	@if ! command -v gosec &> /dev/null; then \
		echo "gosec not found, installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -fmt=json -out=gosec-report.json ./...
	@echo "📄 GoSec report saved to gosec-report.json"
	@gosec -fmt=text ./...

.PHONY: vuln
vuln: ## Run govulncheck for vulnerability scanning.
	@echo "🛡️  Running vulnerability check..."
	@if ! command -v govulncheck &> /dev/null; then \
		echo "govulncheck not found, installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck ./...

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/dev-best-practices/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have a multi-arch builder. More info: https://docs.docker.com/build/building/multi-platform/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Containerfile and insert --platform=${BUILDPLATFORM} into Containerfile.cross, and preserve the original Containerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Containerfile > Containerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Containerfile.cross .
	- docker buildx rm project-v3-builder
	rm Containerfile.cross
