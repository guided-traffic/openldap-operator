.PHONY: build run test test-unit test-integration test-all coverage coverage-ci clean manifests generate fmt vet lint gosec vuln static quality security all-checks docker-build docker-buildx docker-push helm-lint helm-test help tools

# Build variables
BINARY_NAME=manager
BUILD_DIR=bin
COVERAGE_DIR=coverage
HELM_CHART_DIR=deploy/helm/openldap-operator

# Image URL to use all building/pushing image targets
IMG ?= docker.io/hansfischer/openldap-operator:latest

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary
ENVTEST_K8S_VERSION = 1.35.0

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Default target
.PHONY: all
all: build

# Build the operator binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

# Run the operator from your host
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run ./cmd/main.go

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run all tests (excluding integration tests that require Docker)
test:
	@echo "Running all unit tests..."
	$(GOTEST) -v ./api/... ./internal/controller/...

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@$(GOTEST) ./api/... ./internal/controller/... -coverprofile=coverage.out -v 2>&1 | grep -v "does not match go tool version"
	@echo ""
	@echo "Coverage Summary:"
	@$(GOCMD) tool cover -func=coverage.out | tail -1 || echo "No coverage data"

# Run integration tests only (requires Docker)
test-integration:
	@echo "Running integration tests with Docker..."
	./test/run-tests.sh --skip-unit

# Run all tests (unit + integration)
test-all:
	@echo "Running all tests (unit + integration)..."
	./test/run-tests.sh

# Generate test coverage
coverage:
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.txt
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"
	@echo "Coverage summary:"
	@grep "total:" $(COVERAGE_DIR)/coverage.txt

# Generate coverage for CI
coverage-ci:
	@echo "Generating CI coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GOCMD) tool cover -func=coverage.out > $(COVERAGE_DIR)/coverage.txt
	@$(GOCMD) tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated in $(COVERAGE_DIR)/"

# Generate manifests (CRDs, RBAC, etc.)
manifests: controller-gen
	@echo "Generating manifests..."
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=$(HELM_CHART_DIR)/crds

# Generate code (DeepCopy, DeepCopyInto, DeepCopyObject)
generate: controller-gen
	@echo "Generating code..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Static analysis
static:
	@echo "Running static analysis..."
	$(GOVET) ./...
	$(GOFMT) -l .

# Lint the code
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./... --out-format=colored-line-number

# Gosec security scan
gosec:
	@echo "Running gosec security scan..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	@echo "Scanning code for security issues..."
	@gosec -fmt=json -out=gosec-report.json ./... || true
	@echo ""
	@echo "GoSec report saved to gosec-report.json"
	@echo ""
	@echo "Security scan results:"
	@echo "======================"
	@gosec -fmt=text ./...

# Vulnerability check
vuln:
	@echo "Checking for vulnerabilities..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Code quality checks (static + lint + fmt)
quality: static lint fmt

# Security checks (gosec + vuln)
security: gosec vuln

# All checks (quality + security)
all-checks: quality security

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f coverage.out
	rm -f gosec-report.json

# Build docker image
docker-build:
	@echo "Building Docker image..."
	docker build -f Containerfile -t ${IMG} .

# Build and push docker image for cross-platform support
PLATFORMS ?= linux/arm64,linux/amd64
docker-buildx:
	@echo "Building multi-platform Docker image..."
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Containerfile > Containerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Containerfile.cross .
	- docker buildx rm project-v3-builder
	rm Containerfile.cross

# Push docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push ${IMG}

# Helm commands
helm-lint:
	@echo "Linting Helm chart..."
	@which helm > /dev/null || (echo "Helm not found. Please install Helm." && exit 1)
	helm lint $(HELM_CHART_DIR)

helm-test: helm-lint
	@echo "Testing Helm chart..."
	helm template test-release $(HELM_CHART_DIR) > /dev/null
	@echo "Helm chart template test passed"

# Install development tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

##@ Tool Management

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.17.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/controller-gen || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@test -s $(LOCALBIN)/setup-envtest || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the operator binary"
	@echo "  run                - Run the operator from your host"
	@echo "  deps               - Download dependencies"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only"
	@echo "  test-all           - Run all tests (unit + integration)"
	@echo "  coverage           - Generate test coverage report"
	@echo "  coverage-ci        - Generate coverage report for CI"
	@echo "  manifests          - Generate CRDs and RBAC manifests"
	@echo "  generate           - Generate code (DeepCopy methods)"
	@echo "  fmt                - Format the code"
	@echo "  vet                - Run go vet"
	@echo "  static             - Run static analysis"
	@echo "  lint               - Run golangci-lint"
	@echo "  quality            - Run code quality checks (static + lint + fmt)"
	@echo "  security           - Run security checks (gosec + vuln)"
	@echo "  gosec              - Run gosec security scan only"
	@echo "  vuln               - Check for vulnerabilities"
	@echo "  all-checks         - Run all checks (quality + security)"
	@echo "  clean              - Clean build artifacts"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-buildx      - Build multi-platform Docker image"
	@echo "  docker-push        - Push Docker image"
	@echo "  helm-lint          - Lint Helm chart"
	@echo "  helm-test          - Test Helm chart"
	@echo "  tools              - Install development tools"
	@echo "  help               - Show this help"
