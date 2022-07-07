# MicroVM

This folder contains the code and configuration to build a small Linux kernel for the [QEMU microvm][].

## Getting started

Install the dependencies:

```
$ sudo make apt_install
```

Compile the Linux kernel:

```
$ make kernel
```

Download and install `virtiofsd`:

```
$ sudo make virtiofsd
```

## Usage

```
$ make -C .. alpine_bundle
$ make run BUNDLE=/tmp/alpine-bundle/ CID=alpine-qemu
```

[QEMU microvm]: https://qemu.readthedocs.io/en/latest/system/i386/microvm.html
