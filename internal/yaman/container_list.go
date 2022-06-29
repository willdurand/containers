package yaman

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/willdurand/containers/internal/constants"
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/network"
	"github.com/willdurand/containers/internal/yaman/shim"
)

// ContainerListItem contains the data about a container for the user.
type ContainerListItem struct {
	ID           string
	Image        string
	Command      string
	Status       string
	Name         string
	Created      time.Time
	ExposedPorts []network.ExposedPort
}

// ContainerList contains the list of containers to show to the user.
type ContainerList []ContainerListItem

var validId = regexp.MustCompile("[a-z0-9]{32}")

// GetContainerIds returns a list of container IDs managed by Yaman.
func GetContainerIds(rootDir, prefix string) []string {
	var ids []string

	files, err := ioutil.ReadDir(container.GetBaseDir(rootDir))
	if err != nil {
		return ids
	}

	for _, f := range files {
		id := f.Name()
		if validId.MatchString(id) && strings.HasPrefix(id, prefix) {
			ids = append(ids, id)
		}
	}

	return ids
}

// ListContainers returns the list of containers running by default.
//
// Optionally, it can return all containers managed by Yaman (running or not).
// This function does not return `Container` instances but rather data transfer
// objects for a "user interface".
func ListContainers(rootDir string, all bool) (ContainerList, error) {
	var list ContainerList

	for _, id := range GetContainerIds(rootDir, "") {
		shim, err := shim.Load(rootDir, id)
		if err != nil {
			continue
		}

		state, err := shim.GetState()
		if err != nil {
			return nil, err
		}

		if !all && state.State.Status != constants.StateRunning {
			continue
		}

		status := state.State.Status
		if state.Status.Exited() {
			status = fmt.Sprintf(
				"Exited (%d) %s ago",
				state.Status.ExitStatus(),
				units.HumanDuration(time.Since(shim.Container.ExitedAt)),
			)
		}

		list = append(list, ContainerListItem{
			ID:           shim.Container.ID,
			Image:        shim.Container.Image.FQIN(),
			Command:      strings.Join(shim.Container.Command(), " "),
			Status:       status,
			Name:         shim.Container.Opts.Name,
			Created:      shim.Container.CreatedAt,
			ExposedPorts: shim.Container.ExposedPorts,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[j].Created.Before(list[i].Created)
	})

	return list, nil
}
