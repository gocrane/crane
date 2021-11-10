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

LDFLAGS = "-X github.com/gocrane-io/crane/pkg/version.gitVersion=$(GIT_VERSION) \
                      -X github.com/gocrane-io/crane/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
                      -X github.com/gocrane-io/crane/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X github.com/gocrane-io/crane/pkg/version.buildDate=$(BUILDDATE)"

# Images management
REGISTRY ?= ccr.ccs.tencentyun.com
REGISTRY_NAMESPACE ?= kube-orm
REGISTRY_USER_NAME?=""
REGISTRY_PASSWORD?=""

# Image URL to use all building/pushing image targets
MANAGER_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/manager:${GIT_VERSION}"

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
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" rbac:roleName=manager-role crd webhook paths="./vendor/github.com/gocrane-io/api/..." output:crd:artifacts:config=manifests

.PHONY: generate
generate: manifests ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.


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
	@$(GOLANG_LINT) run \
      --timeout 30m \
      --disable-all \
      -E deadcode \
      -E unused \
      -E varcheck \
      -E ineffassign \
      -E goimports \
      -E gofmt \
      -E misspell \
      -E unparam \
      -E unconvert \
      -E govet \
      -E errcheck \
      -E structcheck

.PHONY: test
test: manifests fmt vet ## Run tests.
	go test -race -coverprofile coverage.out -covermode=atomic ./...

##@ Build

.PHONY: all
all: crane-manager

.PHONY: crane-manager
crane-manager: ## Build binary with the manager.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/crane-manager cmd/crane-manager/main.go

.PHONY: images
images: image-crane-manager

.PHONY: image-crane-manager
image-crane-manager: test ## Build docker image with the manager.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=crane-manager -t ${MANAGER_IMG} .

.PHONY: push-images
push-images: push-image-crane-manager

.PHONY: push-image-crane-manager
push-image-crane-manager: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${MANAGER_IMG}

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
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.39.0 ;\
	rm -rf $$GOLANG_LINT_TMP_DIR ;\
	}
GOLANG_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANG_LINT=$(shell which golangci-lint)
endif
