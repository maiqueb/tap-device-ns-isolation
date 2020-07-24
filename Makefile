IMAGE_REGISTRY ?= docker.io
IMAGE_NAME ?= tap-experiment
IMAGE_PULL_POLICY ?= Always
IMAGE_TAG ?= latest

NAMESPACE ?= default

TARGETS = \
	goimports-format \
	goimports-check \
	whitespace-format \
	whitespace-check \
	vet

# tools
GITHUB_RELEASE ?= $(GOBIN)/github-release

# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

# Gather needed source files and directories to create target dependencies
directories=$(filter-out ./ ./vendor/ ./_out/ ./_kubevirtci/ ,$(sort $(dir $(wildcard ./*/))))
all_sources=$(call rwildcard,$(directories),*) $(filter-out $(TARGETS), $(wildcard *))
go_sources=$(call rwildcard,cmd/,*.go) $(call rwildcard,pkg/,*.go) $(call rwildcard,tests/,*.go)

# Configure Go
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
export GO111MODULE=on
export GOFLAGS=-mod=vendor

.ONESHELL:

all: format

format: goimports-format whitespace-format

goimports-check: $(go_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./tests
	touch $@

goimports-format: $(go_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./tests
	touch $@

whitespace-check: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./tests
	touch $@

whitespace-format: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./tests
	touch $@

vet: $(go_sources)
	go vet ./pkg/... ./cmd/... ./tests/...
	touch $@

docker-build:
	docker build -t ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} -f ./cmd/Dockerfile .

docker-push: docker-build
	docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}

vendor:
	go mod tidy
	go mod vendor

.PHONY: \
	all \
	docker-build \
	docker-push \
	vendor
