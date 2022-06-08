# containers

[![CI](https://github.com/willdurand/containers/actions/workflows/ci.yml/badge.svg)](https://github.com/willdurand/containers/actions/workflows/ci.yml)

This is a repository with some code I wrote to **learn** more about containers. It currently contains:

- [`yacr`](./cmd/yacr/README.md): a container runtime that implements the [runtime-spec][]
- [`yacs`](./cmd/yacs/README.md): a container shim with an (HTTP) API

For more information, please refer to the documentation of each sub-project.

Want to give it a quick try? [Open this project in Gitpod](https://gitpod.io/#https://github.com/willdurand/containers)!

## Building this project

The easiest and quickest way to get started is to build all sub-projects:

```
$ make all
```

For non-Gitpod users, add the (absolute path to the) `bin/` directory to your `$PATH` and you should be good to go!

## License

See [`LICENSE.txt`](./LICENSE.txt)

[runtime-spec]: https://github.com/opencontainers/runtime-spec
