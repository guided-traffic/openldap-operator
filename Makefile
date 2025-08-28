# Image URL to use all building/pushing image targets
IMG ?= openldap-operator:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.28.0

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
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit: fmt vet ## Run unit tests only.
	go test ./api/v1/... ./internal/ldap/... ./internal/controller/... -v -cover

.PHONY: test-unit-coverage
test-unit-coverage: fmt vet ## Run unit tests with detailed coverage.
	mkdir -p coverage
	go test ./api/v1/... -coverprofile coverage/api.out -covermode=atomic -v
	go test ./internal/ldap/... -coverprofile coverage/ldap.out -covermode=atomic -v
	go test ./internal/controller/... -coverprofile coverage/controller.out -covermode=atomic -v
	@echo "mode: atomic" > coverage/unit.out
	@grep -h -v "^mode:" coverage/api.out coverage/ldap.out coverage/controller.out >> coverage/unit.out || true
	go tool cover -html=coverage/unit.out -o coverage/unit-coverage.html
	go tool cover -func=coverage/unit.out | tee coverage/unit-coverage.txt
	@echo "Unit test coverage report generated: coverage/unit-coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests with Docker LDAP server.
	@echo "Running integration tests..."
	./test/run-tests.sh

.PHONY: test-integration-external
test-integration-external: ## Run integration tests against external LDAP server (set LDAP_HOST, LDAP_PORT etc).
	@echo "Running integration tests against external LDAP server..."
	./test/run-tests.sh --skip-docker

.PHONY: test-all
test-all: test test-integration ## Run all tests (unit + integration).

.PHONY: test-coverage
test-coverage: manifests generate fmt vet envtest ## Run tests with coverage report.
	mkdir -p coverage
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile coverage/all.out -covermode=atomic -v
	go tool cover -html=coverage/all.out -o coverage/coverage.html
	go tool cover -func=coverage/all.out | tee coverage/coverage.txt
	@echo "Coverage report generated: coverage/coverage.html"

.PHONY: test-coverage-detailed
test-coverage-detailed: test-unit-coverage test-coverage ## Run all tests with detailed coverage analysis.
	@echo "Generating detailed coverage analysis..."
	mkdir -p coverage/reports
	@echo "=== Overall Coverage Summary ===" > coverage/reports/summary.txt
	@echo "" >> coverage/reports/summary.txt
	@echo "Combined Coverage (All Tests):" >> coverage/reports/summary.txt
	@tail -1 coverage/coverage.txt >> coverage/reports/summary.txt
	@echo "" >> coverage/reports/summary.txt
	@echo "Unit Tests Only Coverage:" >> coverage/reports/summary.txt
	@tail -1 coverage/unit-coverage.txt >> coverage/reports/summary.txt
	@echo "" >> coverage/reports/summary.txt
	@echo "=== Detailed Package Coverage ===" >> coverage/reports/summary.txt
	@echo "" >> coverage/reports/summary.txt
	@echo "All Tests by Package:" >> coverage/reports/summary.txt
	@cat coverage/coverage.txt >> coverage/reports/summary.txt
	@echo "" >> coverage/reports/summary.txt
	@echo "Unit Tests by Package:" >> coverage/reports/summary.txt
	@cat coverage/unit-coverage.txt >> coverage/reports/summary.txt
	@echo ""
	@echo "Detailed coverage analysis completed!"
	@echo "- HTML reports: coverage/coverage.html, coverage/unit-coverage.html"
	@echo "- Text reports: coverage/coverage.txt, coverage/unit-coverage.txt"
	@echo "- Summary: coverage/reports/summary.txt"
	@echo ""
	@cat coverage/reports/summary.txt

.PHONY: coverage-analysis
coverage-analysis: ## Run detailed coverage analysis with recommendations.
	./scripts/coverage-analysis.sh

.PHONY: test-all-coverage
test-all-coverage: test-coverage-detailed coverage-analysis ## Run all tests with comprehensive coverage analysis and recommendations.

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

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have a multi-arch builder. More info: https://docs.docker.com/build/building/multi-platform/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Containerfile and insert --platform=${BUILDPLATFORM} into Containerfile.cross, and preserve the original Containerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Containerfile > Containerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Containerfile.cross .
	- docker buildx rm project-v3-builder
	rm Containerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f deploy/helm/openldap-operator/crds/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kubectl delete --ignore-not-found=$(ignore-not-found) -f deploy/helm/openldap-operator/crds/

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config using Helm.
	helm upgrade --install openldap-operator deploy/helm/openldap-operator --set image.repository=${IMG%:*} --set image.tag=${IMG#*:}

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config using Helm.
	helm uninstall openldap-operator --ignore-not-found

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.19.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
