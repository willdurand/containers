package container

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/runtime"
	"github.com/willdurand/containers/internal/yacr/ipc"
)

type YacrContainer struct {
	*runtime.BaseContainer
}

func New(rootDir string, id string, bundleDir string) (*YacrContainer, error) {
	base, err := runtime.New(rootDir, id, bundleDir)
	return &YacrContainer{base}, err
}

func Load(rootDir string, id string) (*YacrContainer, error) {
	base, err := runtime.Load(rootDir, id)
	return &YacrContainer{base}, err
}

func LoadWithBundleConfig(rootDir string, id string) (*YacrContainer, error) {
	base, err := runtime.LoadWithBundleConfig(rootDir, id)
	return &YacrContainer{base}, err
}

func LoadFromContainer(BaseDir string, id string) (*YacrContainer, error) {
	container := &YacrContainer{
		BaseContainer: &runtime.BaseContainer{},
	}

	c, err := runtime.LoadWithBundleConfig(BaseDir, id)
	if err != nil {
		return container, err
	}

	container.BaseContainer.Spec = c.Spec
	container.BaseContainer.State = c.State
	// See: https://github.com/opencontainers/runtime-spec/blob/a3c33d663ebc56c4d35dbceaa447c7bf37f6fab3/runtime.md#state
	container.State.Pid = os.Getpid()
	container.CreatedAt = c.CreatedAt
	container.BaseDir = c.BaseDir
	container.StateFilePath = ""

	return container, nil
}

func (c *YacrContainer) GetInitSockAddr(mustExist bool) (string, error) {
	initSockAddr := filepath.Join(c.BaseDir, "init.sock")
	return initSockAddr, ipc.EnsureValidSockAddr(initSockAddr, mustExist)
}

func (c *YacrContainer) GetSockAddr(mustExist bool) (string, error) {
	sockAddr := filepath.Join(c.BaseDir, "ipc.sock")
	return sockAddr, ipc.EnsureValidSockAddr(sockAddr, mustExist)
}

func (c *YacrContainer) ExecuteHooks(name string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	hooks := map[string][]runtimespec.Hook{
		"Prestart":        c.Spec.Hooks.Prestart,
		"CreateRuntime":   c.Spec.Hooks.CreateRuntime,
		"CreateContainer": c.Spec.Hooks.CreateContainer,
		"StartContainer":  c.Spec.Hooks.StartContainer,
		"Poststart":       c.Spec.Hooks.Poststart,
		"Poststop":        c.Spec.Hooks.Poststop,
	}[name]

	if len(hooks) == 0 {
		logrus.WithFields(logrus.Fields{
			"id":    c.ID(),
			"name:": name,
		}).Debug("no hooks")

		return nil
	}

	logrus.WithFields(logrus.Fields{
		"id":    c.ID(),
		"name:": name,
		"state": c.State,
		"hooks": hooks,
	}).Debug("executing hooks")

	s, err := json.Marshal(c.State)
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		var stdout, stderr bytes.Buffer

		cmd := exec.Cmd{
			Path:   hook.Path,
			Args:   hook.Args,
			Env:    hook.Env,
			Stdin:  bytes.NewReader(s),
			Stdout: &stdout,
			Stderr: &stderr,
		}

		if err := cmd.Run(); err != nil {
			logrus.WithFields(logrus.Fields{
				"id":     c.ID(),
				"name:":  name,
				"error":  err,
				"stderr": stderr.String(),
				"stdout": stdout.String(),
			}).Error("failed to execute hooks")

			return fmt.Errorf("failed to execute %s hook '%s': %w", name, cmd.String(), err)
		}
	}

	return nil
}
