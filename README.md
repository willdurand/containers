# containers

[![CI](https://github.com/willdurand/containers/actions/workflows/ci.yml/badge.svg)](https://github.com/willdurand/containers/actions/workflows/ci.yml)

<p align="center">
  <img src="./docs/yaman.svg" />
</p>

This is a repository with some code I wrote to **learn** more about containers. It currently contains:

- [`yacr`](./cmd/yacr/README.md): a container runtime that implements the [runtime-spec][]
- [`yacs`](./cmd/yacs/README.md): a container shim with an (HTTP) API
- [`yaman`](./cmd/yaman/README.md): a container manager that leverages the two previous tools

For more information, please refer to the documentation of each sub-project.

Want to give it a quick try? [Open this project in Gitpod](https://gitpod.io/#https://github.com/willdurand/containers)!

## Building this project

This project requires a Linux environment. You can use Gitpod as mentioned above or [Vagrant][].

The easiest and quickest way to get started is to build all sub-projects:

```
$ make all
$ sudo make install
```

### Vagrant

This project can be used with [Vagrant][] to set up a Linux virtual machine. This is recommended as opposed to trying out the different tools on your actual Linux system.

```
$ vagrant up && vagrant ssh
```

From there, you can `cd /vagrant` and follow the instruction to build the project (previous section).

### bash auto-completion on Linux

You can also enable completion for `yacr` and `yaman`.

You must have the `bash-completion` package installed first and you should possibly source `/usr/share/bash-completion/bash_completion` in your bash configuration (`~/.bashrc`). When `type _init_completion` in your shell returns some content, you're all set.

You can now install the completion files:

```
$ sudo make install_completion
```

After reloading your shell, `yacr` and `yaman` autocompletion should be working.

## License

See [`LICENSE.txt`](./LICENSE.txt)

[runtime-spec]: https://github.com/opencontainers/runtime-spec
[vagrant]: https://www.vagrantup.com/
