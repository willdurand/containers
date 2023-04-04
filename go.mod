module github.com/willdurand/containers

go 1.18

require github.com/spf13/cobra v1.4.0

require (
	github.com/artyom/untar v1.0.1
	github.com/creack/pty v1.1.18
	github.com/docker/docker v20.10.24+incompatible
	github.com/docker/go-units v0.4.0
	github.com/google/uuid v1.3.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runtime-spec v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/sevlyar/go-daemon v0.1.5
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467
)

require (
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	gotest.tools/v3 v3.2.0 // indirect
)

require (
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a
)
