MAKEFLAGS += --warn-undefined-variables

.DEFAULT_GOAL := help

# Disable implicit rules.
.SUFFIXES:

git_hash  := $(shell git rev-parse --short HEAD)
build_dir := build

build: ## build binaries
	@mkdir -p $(build_dir)
	cd yacr && go build -ldflags "-X github.com/willdurand/containers/yacr/version.GitCommit=$(git_hash)" -o ../$(build_dir)/yacr
.PHONY: build

help: ## show this help message
help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
