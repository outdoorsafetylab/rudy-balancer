TOOLCHAIN ?= .tool
GOLANGCI_LINT := $(TOOLCHAIN)/bin/golangci-lint

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLCHAIN)/bin v1.24.0

clean/golangci-lint:
	rm -f $(GOLANGCI_LINT)

.PHONY: clean/golangci-lint
