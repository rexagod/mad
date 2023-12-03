# Variables are declared in the order in which they occur.
VALE_VERSION ?= 3.1.0
VALE_ARCH ?= Linux_64-bit # macOS_arm64 for Apple Silicon.
ASSETS_DIR ?= assets/
GO ?= go
MARKDOWNFMT_VERSION ?= v3.1.0
GOLANGCI_LINT_VERSION ?= v1.54.2
MD_FILES = $(shell find . \( -type d -name 'vendor' -o -type d -name $(patsubst %/,%,$(patsubst ./%,%,$(ASSETS_DIR))) \) -prune -o -type f -name "*.md" -print)
GO_FILES = $(shell find . -type f -name "*.go")
OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH ?= $(shell $(GO) env GOARCH)
COMMON = github.com/prometheus/common
VERSION = $(shell cat VERSION)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
RUNNER = $(shell id -u -n)@$(shell hostname)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
BUILD_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "latest")

all: lint mad

.PHONY: setup-dependencies
setup-dependencies:
	@# Setup vale.
	@wget https://github.com/errata-ai/vale/releases/download/v$(VALE_VERSION)/vale_$(VALE_VERSION)_$(VALE_ARCH).tar.gz && \
	mkdir -p assets && tar -xvzf vale_$(VALE_VERSION)_$(VALE_ARCH).tar.gz -C assets && \
	chmod +x $(ASSETS_DIR)vale && \
	mkdir -p /tmp/.vale/styles && \
	$(ASSETS_DIR)vale sync
	@# Setup markdownfmt.
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) install github.com/Kunde21/markdownfmt/v3/cmd/markdownfmt@$(MARKDOWNFMT_VERSION)
	@# Setup golangci-lint.
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@# Setup controller-gen.
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1

.PHONY: manifests
manifests:
	@controller-gen paths="./..." crd:trivialVersions=true output:crd:artifacts:config=/tmp/mad-manifests
	@controller-gen paths="./..." rbac:roleName="mad-controller" output:rbac:artifacts:config=/tmp/mad-manifests
	@mv "/tmp/mad-manifests/mad.instrumentation.k8s-sigs.io_metricsanomalydetectorresources.yaml" "manifests/custom-resource-definition.yaml"
	@mv "/tmp/mad-manifests/role.yaml" "manifests/cluster-role.yaml"
	@rm -rf "/tmp/mad-manifests"

.PHONY: deploy
deploy:
	@docker build -t mad:$(BUILD_TAG) .
	@kind load docker-image mad:$(BUILD_TAG)
	@kubectl apply -f manifests/


.PHONY: test-unit
test-unit:
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) test -v -race $(shell $(GO) list ./... | grep -v $(E2E_TEST_PKG))

.PHONY: test
test: test-unit

.PHONY: clean
clean:
	@git clean -fxd

vale: .vale.ini $(MD_FILES)
	@$(ASSETS_DIR)vale $(MD_FILES)

markdownfmt: $(MD_FILES)
	@test -z "$(shell markdownfmt -l $(MD_FILES))" || (echo "\033[0;31mThe following files need to be formatted with 'markdownfmt -w -gofmt':" $(shell markdownfmt -l $(MD_FILES)) "\033[0m" && exit 1)

.PHONY: lint-md
lint-md: vale markdownfmt

gofmt: $(GO_FILES)
	@test -z "$(shell gofmt -l $(GO_FILES))" || (echo "\033[0;31mThe following files need to be formatted with 'gofmt -w':" $(shell gofmt -l $(GO_FILES)) "\033[0m" && exit 1)

golangci-lint: $(GO_FILES)
	@golangci-lint run

.PHONY: lint-go
lint-go: gofmt golangci-lint

.PHONY: lint
lint: lint-md lint-go

markdownfmt-fix: $(MD_FILES)
	@for file in $(MD_FILES); do markdownfmt -w -gofmt $$file || exit 1; done

.PHONY: lint-md-fix
lint-md-fix: vale markdownfmt-fix

gofmt-fix: $(GO_FILES)
	@gofmt -w . || exit 1

.PHONY: lint-go-fix
lint-go-fix: gofmt-fix golangci-lint

.PHONY: lint-fix
lint-fix: lint-md-fix lint-go-fix

mad: $(GO_FILES)
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -a -installsuffix cgo -ldflags "-s -w \
	-X ${COMMON}/version.Version=v${VERSION} \
	-X ${COMMON}/version.Revision=${GIT_COMMIT} \
	-X ${COMMON}/version.Branch=${BRANCH} \
	-X ${COMMON}/version.BuildUser=${RUNNER} \
	-X ${COMMON}/version.BuildDate=${BUILD_DATE}" \
	-o $@
