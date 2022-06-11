package yacr

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yacr/container"
)

type ContainerListItem struct {
	ID         string
	Status     string
	CreatedAt  time.Time
	PID        int
	BundlePath string
}

type ContainerList []ContainerListItem

func List(rootDir string) (ContainerList, error) {
	var list ContainerList

	files, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return list, fmt.Errorf("failed to read root directory: %w", err)
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		container, err := container.Load(rootDir, f.Name())
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"id":    f.Name(),
				"error": err,
			}).Debug("failed to load container")
			continue
		}

		state := container.State()

		pid := state.Pid
		if container.IsStopped() {
			pid = 0
		}

		list = append(list, ContainerListItem{
			ID:         container.ID(),
			Status:     state.Status,
			CreatedAt:  container.CreatedAt(),
			PID:        pid,
			BundlePath: state.Bundle,
		})
	}

	return list, nil
}
