PROTOS := $(wildcard *.proto) $(wildcard */*.proto) $(wildcard */*/*.proto)

PBGO := $(PROTOS:.proto=.pb.go)
GOSRCS := go.mod $(wildcard *.go) $(wildcard */*.go) $(wildcard */*/*.go)

EXEC := serviced

BUILD_TIME ?= $(shell date +'%s')
GIT_HASH ?= $(shell git rev-parse --short HEAD)
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "")

VARS :=
VARS += BuildTime=$(BUILD_TIME)
VARS += GitHash=$(GIT_HASH)
VARS += GitTag=$(GIT_TAG)
LDFLAGS := $(addprefix -X version.,$(VARS))

all: $(EXEC)

include .make/golangci-lint.mk
include .make/protoc.mk
include .make/protoc-gen-go.mk
include .make/watcher.mk
include .make/docker.mk

watch: $(PBGO) $(WEBINDEX) $(WATCHER) tidy
	$(realpath $(WATCHER)) -c local

tidy: $(PBGO)
	go mod tidy

lint: $(GOLANGCI_LINT)
	$(realpath $(GOLANGCI_LINT)) run

$(EXEC): $(PBGO) $(GOSRCS)
	go mod tidy
	go build -ldflags="$(LDFLAGS)" -o $@

clean/proto:
	rm -f $(PBGO)

clean: clean/golangci-lint clean/protoc clean/protoc-gen-go clean/proto clean/watcher
	rm -f go.sum
	rm -f $(EXEC)

.PHONY: all tidy lint clean test
