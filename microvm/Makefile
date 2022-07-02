# Disabled because there are many undefined variables in the kernel's
# `scripts/Makefile.build` file
# MAKEFLAGS += --warn-undefined-variables

.DEFAULT_GOAL := help

# Disable implicit rules.
.SUFFIXES:

BUNDLE    ?= /tmp/alpine-bundle
CID       ?=
CMD       ?= /bin/sh
DEBUG     ?=
BUILD_DIR ?= ./build

kernel_version      = linux-5.15.47
kernel_archive_file = $(kernel_version).tar.xz
kernel_download_url = https://cdn.kernel.org/pub/linux/kernel/v5.x/$(kernel_archive_file)

build_dir      = $(BUILD_DIR)
kernel_src_dir = $(build_dir)/$(kernel_version)
kernel_archive = $(build_dir)/$(kernel_archive_file)
vmlinux        = $(kernel_src_dir)/vmlinux
rootfs         = $(BUNDLE)/rootfs
init           = $(rootfs)/sbin/init
virtiofsd      = $(build_dir)/virtiofsd
virtiofsd_sock = $(build_dir)/virtiofsd.sock
qemu_mem       = 512m

werrors += -Wall -Wextra -Werror
werrors += -Wformat=2
werrors += -Wno-null-pointer-arithmetic
cflags  += -static
cflags  += $(werrors)

$(kernel_archive):
	@mkdir -p $(build_dir)
	curl -s $(kernel_download_url) -o $@

$(kernel_src_dir): $(kernel_archive)
	tar -C $(build_dir) -xf $(kernel_archive)

$(kernel_src_dir)/.config: config-x86_64
	cp $< $@
	$(MAKE) -C $(kernel_src_dir) olddefconfig

$(vmlinux): $(kernel_src_dir) $(kernel_src_dir)/.config
	cd $(kernel_src_dir) && for patch_file in $(PWD)/patches/*.patch; do patch -N -p0 < "$$patch_file" || true; done
	$(MAKE) -C $(kernel_src_dir) -j$(shell nproc)

kernel: ## build the kernel (vmlinux)
kernel: $(vmlinux)
.PHONY: kernel

apt_install: ## run `apt-get install -y` with a pre-defined list of dependencies
	add-apt-repository -y ppa:canonical-server/server-backports
	apt-get update -yqq
	apt-get install -y bison flex libelf-dev qemu-system
.PHONY: apt_install

init: ## build `init(1)`
init: $(init)
.PHONY: init

$(init): init.c
	rm -f $(init)
	gcc $(cflags) -o $@ $<

virtiofsd: ## download virtiofsd, which we use to "mount" a rootfs in a VM
virtiofsd: $(virtiofsd)
.PHONY: virtiofsd

$(virtiofsd):
	curl -s -L https://gitlab.com/virtio-fs/virtiofsd/-/jobs/artifacts/main/download?job=publish -o $(build_dir)/virtiofsd.zip
	cd $(build_dir) && unzip virtiofsd.zip
	cp $(build_dir)/target/x86_64-unknown-linux-musl/release/virtiofsd $(virtiofsd)

run: ## run a container (inside a QEMU VM)
run: $(kernel) $(init) $(virtiofsd)
	@sudo $(virtiofsd) --syslog \
		--socket-path=$(virtiofsd_sock) \
		--shared-dir=$(rootfs) \
		--socket-group=$(shell groups) \
		--cache=never \
		--sandbox=none \
		&
	qemu-system-x86_64 \
		-M microvm \
		-m $(qemu_mem) \
		-no-acpi -no-reboot -no-user-config -nodefaults -nographic \
		-chardev stdio,id=virtiocon0 \
		-device virtio-serial-device \
		-device virtconsole,chardev=virtiocon0 \
		-chardev socket,id=virtiofs0,path=$(virtiofsd_sock) \
		-device vhost-user-fs-device,queue-size=1024,chardev=virtiofs0,tag=/dev/root \
		-object memory-backend-file,id=mem,size=$(qemu_mem),mem-path=/dev/shm,share=on \
		-numa node,memdev=mem \
		-kernel $(vmlinux) \
		-append "quiet reboot=t console=hvc0 rootfstype=virtiofs root=/dev/root rw MV_DEBUG=$(DEBUG) MV_HOSTNAME=$(CID) MV_INIT=$(CMD)"
.PHONY: run

help: ## show this help message
help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
