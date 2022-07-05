MAKEFLAGS += --warn-undefined-variables

.DEFAULT_GOAL := help

# Disable implicit rules.
.SUFFIXES:

git_hash      := $(shell git rev-parse --short HEAD)
bin_dir       := $(CURDIR)/bin
binaries      := yacr yacs yaman microvm
binaries_comp := yacr yaman

go_build_flags := -ldflags "-X github.com/willdurand/containers/internal/version.GitCommit=$(git_hash)"

all: ## build all binaries
all: $(binaries)
.PHONY: all

install: ## install the binaries on the system using symlinks (sudo required)
	@for binary in $(binaries); do \
		ln -fs "$(bin_dir)/$$binary" "/usr/local/bin/$$binary"; \
	done
.PHONY: install

install_completion: ## generate and install the completion files (sudo required)
	@for binary in $(binaries_comp); do \
		"$(bin_dir)/$$binary" completion bash | tee "/etc/bash_completion.d/$$binary" > /dev/null; \
	done
.PHONY: install_completion

yacr: ## build the container runtime
	@mkdir -p $(bin_dir)
	cd cmd/$@ && go build $(go_build_flags) -o "$(bin_dir)/$@"
.PHONY: yacr

yacs: ## build the container shim
	@mkdir -p $(bin_dir)
	cd cmd/$@ && go build $(go_build_flags) -o "$(bin_dir)/$@"
.PHONY: yacs

yaman: ## build the container manager
	@mkdir -p $(bin_dir)
	cd cmd/$@ && go build $(go_build_flags) -o "$(bin_dir)/$@"
.PHONY: yaman

microvm: ## build another (experimental) runtime
microvm: cmd/microvm/init microvm/build/vmlinux
	@mkdir -p $(bin_dir)
	cd cmd/$@ && go build $(go_build_flags) -o "$(bin_dir)/$@"
.PHONY: microvm

cmd/microvm/init: microvm/init.c
	$(MAKE) -C microvm init
	cp microvm/build/init $@

microvm/build/vmlinux: MAKEFLAGS=
microvm/build/vmlinux:
	$(MAKE) -C microvm kernel

alpine_bundle: ## create a rootless bundle (for testing purposes)
	rm -rf /tmp/alpine-bundle/rootfs
	mkdir -p /tmp/alpine-bundle/rootfs
	docker export $$(docker create alpine) | tar -C /tmp/alpine-bundle/rootfs -xvf -
	yacr spec --bundle /tmp/alpine-bundle --rootless
.PHONY: alpine_bundle

hello_world_image:
	cd extras/docker/hello-world/ && \
	zig cc -target x86_64-linux-musl -static hello.c -o hello && \
	docker build -t willdurand/hello-world .
.PHONY: hello_world_image

apt_install:
	$(MAKE) -C microvm apt_install
	apt-get install -y fuse-overlayfs slirp4netns uidmap netcat jq
	which runc || apt-get install -y runc
	which tap || (apt-get install -y nodejs npm && npm install -g tap)
.PHONY: apt_install

help: ## show this help message
help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
