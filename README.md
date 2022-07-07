# containers

[![CI](https://github.com/willdurand/containers/actions/workflows/ci.yml/badge.svg)](https://github.com/willdurand/containers/actions/workflows/ci.yml)

This is a repository with some code I wrote to **learn** more about containers. It currently contains:

- [`yacr`](./cmd/yacr/README.md): a container runtime that implements the [runtime-spec][]
- [`yacs`](./cmd/yacs/README.md): a container shim with an (HTTP) API
- [`yaman`](./cmd/yaman/README.md): a container manager that leverages the two previous tools
- [`microvm`][microvm]: a container runtime that uses micro Virtual Machines (VMs)

For more information, please refer to the documentation of each sub-project.

[![asciicast](https://asciinema.org/a/vdC2zxvyHSubTHuAPDt3g2T21.svg)](https://asciinema.org/a/vdC2zxvyHSubTHuAPDt3g2T21)

Want to give it a quick try? [Open this project in Gitpod](https://gitpod.io/#https://github.com/willdurand/containers)!

## Building this project

This project requires a Linux environment and the following dependencies:

- `fuse-overlayfs` for rootless containers
- `uidmap` for rootless containers
- `slirp4netns` for the network layer (rootfull and rootless containers)
- `bats`, `netcat`, `jq` and `runc` for the integration tests
- `tap` (or `node-tap`) for the OCI comformance tests

For the [`microvm`][microvm] runtime, this project also requires:

- `bison`, `flex`, `libelf-dev` to build the Linux kernel
- `qemu-system`

You should use Gitpod as mentioned in the previously or [Vagrant][]. It might not be a good idea to run this project on your actual machine.

The easiest and quickest way to get started is to build all sub-projects:

```console
$ make all
$ sudo make install
```

Optional step: run `sudo make install_completion` to enable `bash` auto-completion.

### Vagrant

This project can be used with [Vagrant][] to set up a Linux virtual machine. This is recommended as opposed to trying out the different tools on your actual Linux system.

```console
$ vagrant up && vagrant ssh
```

From there, you can `cd /vagrant` and follow the instruction to build the project (previous section).

### bash auto-completion on Linux

You can also enable completion for `yacr` and `yaman`.

You must have the `bash-completion` package installed first and you should possibly source `/usr/share/bash-completion/bash_completion` in your bash configuration (`~/.bashrc`). When `type _init_completion` in your shell returns some content, you're all set.

You can now install the completion files:

```console
$ sudo make install_completion
```

After reloading your shell, `yacr` and `yaman` autocompletion should be working.

## License

See [`LICENSE.txt`](./LICENSE.txt)

[runtime-spec]: https://github.com/opencontainers/runtime-spec
[vagrant]: https://www.vagrantup.com/
[microvm]: ./cmd/microvm/README.md
