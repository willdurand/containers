# microvm

## Getting started

Install the dependencies:

```
$ sudo make apt_install
```

Compile the Linux kernel and download `virtiofsd`:

```
$ make kernel virtiofsd
```

## Usage

```
$ make -C .. alpine_bundle
$ make run BUNDLE=/tmp/alpine-bundle/ CID=alpine-qemu
```