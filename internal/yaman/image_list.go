package yaman

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/willdurand/containers/internal/yaman/image"
)

type ImageListItem struct {
	Registry string
	Name     string
	Version  string
	Created  time.Time
	Pulled   time.Time
}

type ImageList []ImageListItem

func ListImages(rootDir string) (ImageList, error) {
	var list ImageList

	imagesDir := image.GetBaseDir(rootDir)

	// This is what we are going to traverse:
	//
	// /run/yaman/images
	// └── docker.io
	//     └── library
	//         └─── alpine
	//             └── latest
	//                 ├── blobs
	//                 │   ├── config.json
	//                 │   └── sha256:2408cc74d12b6cd092bb8b516ba7d5e290f485d3eb9672efc00f0583730179e8
	//                 ├── layers
	//                 │   └── 24302eb7d9085da80f016e7e4ae55417e412fb7e0a8021e95e3b60c67cde557d
	//                 └── manifest.json
	//
	// There is only one image with the following properties:
	//
	// - hostname: docker.io
	// - user:     library
	// - image:    alpine
	// - version:  latest
	//
	hostnames, err := ioutil.ReadDir(imagesDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return list, err
	}

	for _, hostname := range hostnames {
		hostnameDir := filepath.Join(imagesDir, hostname.Name())
		users, err := ioutil.ReadDir(hostnameDir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return list, err
		}

		for _, user := range users {
			userDir := filepath.Join(hostnameDir, user.Name())
			images, err := ioutil.ReadDir(userDir)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return list, err
			}

			for _, i := range images {
				imageName := i.Name()
				imageDir := filepath.Join(userDir, imageName)
				versions, err := ioutil.ReadDir(imageDir)
				if err != nil && !errors.Is(err, fs.ErrNotExist) {
					return list, err
				}

				for _, version := range versions {
					fullyQualifiedImageName := strings.Join([]string{
						hostname.Name(),
						user.Name(),
						imageName,
					}, "/") + ":" + version.Name()

					img, err := image.New(rootDir, fullyQualifiedImageName)
					if err != nil {
						return list, err
					}

					config, err := img.Config()
					if err != nil {
						return list, err
					}

					list = append(list, ImageListItem{
						Registry: img.Hostname,
						Name:     img.Name,
						Version:  img.Version,
						Created:  *config.Created,
						Pulled:   version.ModTime(),
					})
				}
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[j].Pulled.Before(list[i].Pulled)
	})

	return list, nil
}
