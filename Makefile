MAKEFLAGS += --warn-undefined-variables

.DEFAULT_GOAL := help

# Disable implicit rules.
.SUFFIXES:

git_hash  := $(shell git rev-parse --short HEAD)
bin_dir   := bin

go_build_flags := -ldflags "-X github.com/willdurand/containers/internal/version.GitCommit=$(git_hash)"

all: ## build all binaries
all: yacr yacs
.PHONY: all

yacr: ## build the container runtime
	@mkdir -p $(bin_dir)
	cd cmd/$@ && go build $(go_build_flags) -o "../../$(bin_dir)/$@"
.PHONY: yacr

yacs: ## build the container shim
	@mkdir -p $(bin_dir)
	cd cmd/$@ && go build $(go_build_flags) -o "../../$(bin_dir)/$@"
.PHONY: yacs

alpine_bundle: ## create a rootless bundle (for testing purposes)
	rm -rf /tmp/alpine-bundle/rootfs
	mkdir -p /tmp/alpine-bundle/rootfs
	docker export $$(docker create alpine) | tar -C /tmp/alpine-bundle/rootfs -xvf -
	cd /tmp/alpine-bundle && runc spec --rootless
.PHONY: alpine_bundle

help: ## show this help message
help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
