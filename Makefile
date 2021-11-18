# Recipes
all: help

.PHONY: init 
init: ## Init stable version
	GO111MODULE=on go mod vendor

.PHONY: test
test: ## Run tests
	@CGO_ENABLED=0 go test -mod vendor -test.v -cover ./...

.PHONY: lint
lint: ## Lint the files
	@golangci-lint --version; \
	golangci-lint linters; \
	CGO_ENABLED=0 golangci-lint run -v

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; \
	{printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
		

.PHONY: all test lint init