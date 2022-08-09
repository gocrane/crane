GOOS ?= $(shell go env GOOS)

# Git information
GIT_VERSION ?= $(shell git describe --tags --always)
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif
BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS = "-X github.com/gocrane/crane/pkg/version.gitVersion=$(GIT_VERSION) \
                      -X github.com/gocrane/crane/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
                      -X github.com/gocrane/crane/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X github.com/gocrane/crane/pkg/version.buildDate=$(BUILDDATE)"

# Images management
REGISTRY ?= docker.io
REGISTRY_NAMESPACE ?= gocrane
REGISTRY_USER_NAME?=""
REGISTRY_PASSWORD?=""

# Image URL to use all building/pushing image targets
MANAGER_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/craned:${GIT_VERSION}"
AGENT_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/crane-agent:${GIT_VERSION}"
ADAPTER_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/metric-adapter:${GIT_VERSION}"
DASHBOARD_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/dashboard:${GIT_VERSION}"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

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
	go mod vendor; \
    $(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" rbac:roleName=manager-role crd webhook paths="./vendor/github.com/gocrane/api/..." output:crd:artifacts:config=deploy/manifests; \
    rm -rf vendor

.PHONY: go-mockgen
go-mockgen: mockgen ## Run go mockgen to gen mock code.
	go generate ./...

.PHONY: generate
generate: manifests go-mockgen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

# Run mod tidy against code
.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint: golangci-lint  ## Run golang lint against code
	@$(GOLANG_LINT) run ./...

.PHONY: test
test: fmt vet lint ## Run tests.
	go test -coverprofile coverage.out -covermode=atomic ./...

.PHONY: echoLDFLAGS
echoLDFLAGS:
	@echo $(LDFLAGS)

.PHONY: build
build: craned crane-agent metric-adapter

.PHONY: all
all: generate test craned  crane-agent metric-adapter
.PHONY: craned
craned: ## Build binary with the crane manager.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/craned cmd/craned/main.go

.PHONY: crane-agent
crane-agent: ## Build binary with the crane agent.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/crane-agent cmd/crane-agent/main.go

.PHONY: metric-adapter
metric-adapter: ## Build binary with the metric adapter.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/metric-adapter cmd/metric-adapter/main.go

.PHONY: images
images: image-craned image-crane-agent image-metric-adapter image-dashboard

.PHONY: image-craned
image-craned: ## Build docker image with the crane manager.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=craned -t ${MANAGER_IMG} .

.PHONY: image-dashboard
image-dashboard: ## Build docker image with the crane dashboard.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=web -t ${DASHBOARD_IMG} ./pkg/web

.PHONY: image-crane-agent
image-crane-agent: ## Build docker image with the crane agent.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=crane-agent -t ${AGENT_IMG} .

.PHONY: image-metric-adapter
image-metric-adapter: ## Build docker image with the metric adapter.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=metric-adapter -t ${ADAPTER_IMG} .

.PHONY: push-images
push-images: push-image-craned push-image-crane-agent push-image-metric-adapter push-image-dashboard

.PHONY: push-image-craned
push-image-craned: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${MANAGER_IMG}

.PHONY: push-image-dashboard
push-image-dashboard: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DASHBOARD_IMG}

.PHONY: push-image-crane-agent
push-image-crane-agent: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${AGENT_IMG}

.PHONY: push-image-metric-adapter
push-image-metric-adapter: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${ADAPTER_IMG}

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

golangci-lint:
ifeq (, $(shell which golangci-lint))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GOLANG_LINT_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOLANG_LINT_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0 ;\
	rm -rf $$GOLANG_LINT_TMP_DIR ;\
	}
GOLANG_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANG_LINT=$(shell which golangci-lint)
endif

goimports:
ifeq (, $(shell which goimports))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GO_IMPORTS_TMP_DIR=$$(mktemp -d) ;\
	cd $$GO_IMPORTS_TMP_DIR ;\
	go mod init tmp ;\
	go get golang.org/x/tools/cmd/goimports@v0.1.7 ;\
	rm -rf $$GO_IMPORTS_TMP_DIR ;\
	}
GO_IMPORTS=$(shell go env GOPATH)/bin/goimports
else
GO_IMPORTS=$(shell which goimports)
endif

mockgen:
ifeq (, $(shell which mockgen))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GO_MOCKGEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$GO_MOCKGEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/golang/mock/mockgen@v1.5.0 ;\
	go install github.com/golang/mock/mockgen ;\
	rm -rf $$GO_MOCKGEN_TMP_DIR ;\
	}
GO_MOCKGEN=$(shell go env GOPATH)/bin/mockgen
else
GO_MOCKGEN=$(shell which mockgen)
endif
