# Yet Another (container) MANager

Yaman is a daemon-less container manager inspired by [Docker][] and [Podman][].

## Commands

### `yaman image`

Manage OCI images.

#### `yaman image list`

```
$ sudo yaman image list
NAME             TAG         CREATED                PULLED         REGISTRY
library/alpine   latest      2022-05-23T19:19:31Z   34 hours ago   docker.io
library/redis    latest      2022-06-08T18:34:43Z   47 hours ago   docker.io
```

[docker]: https://docs.docker.com/reference/
[podman]: https://docs.podman.io/en/latest/
