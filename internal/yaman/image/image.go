package image

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/willdurand/containers/internal/yaman/network"
)

// Image represents a OCI image.
type Image struct {
	Hostname string
	Name     string
	Version  string
	BaseDir  string
	Manifest *imagespec.Manifest
	Config   *imagespec.Image
}

const defaultImageVersion = "latest"

// imageNamePattern is the regular expression used to validate an OCI image
// name according to the OCI specification.
var imageNamePattern = regexp.MustCompile("^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$")

// New creates a new image given a directory (to store the image) and the name
// of the image, which must be fully qualified.
func New(rootDir, name string) (*Image, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("image name must be fully qualified")
	}
	hostName := parts[0]
	userName := parts[1]

	parts = strings.Split(parts[2], ":")
	imageName := userName + "/" + parts[0]
	if !isNameValid(imageName) {
		return nil, fmt.Errorf("invalid image name")
	}

	imageVersion := defaultImageVersion
	if len(parts) > 1 && parts[1] != "" {
		imageVersion = parts[1]
	}

	img := &Image{
		Hostname: hostName,
		Name:     imageName,
		Version:  imageVersion,
		BaseDir:  filepath.Join(GetBaseDir(rootDir), hostName, imageName, imageVersion),
	}

	return img, nil
}

// GetBaseDir returns the base directory where all images are stored (locally).
func GetBaseDir(rootDir string) string {
	return filepath.Join(rootDir, "images")
}

// FQIN returns the Fully Qualified Image Name of an image.
func (i *Image) FQIN() string {
	return i.Hostname + "/" + i.Name + ":" + i.Version
}

// LayerDirs returns a list of absolute (directory) paths pointing to the
// different layers of the image. This list is ordered so that the last layer
// directory in the list is the lowest directory (which matches the expectation
// of `lowerdir` in Overlay).
func (i *Image) LayerDirs() []string {
	var dirs []string
	// Note: The order of lower directories is the rightmost is the lowest, thus
	// the upper directory is on top of the first directory in the left-to-right
	// list of lower directories; NOT on top of the last directory in the list,
	// as the order might seem to suggest.
	//
	// Source: https://wiki.archlinux.org/title/Overlay_filesystem
	for idx := len(i.Config.RootFS.DiffIDs) - 1; idx >= 0; idx-- {
		digest := i.Config.RootFS.DiffIDs[idx]
		dirs = append(dirs, filepath.Join(i.LayersDir(), digest.Encoded()))
	}

	return dirs
}

// ManifestFilePath returns the path to the `manifest.json` file.
func (i *Image) ManifestFilePath() string {
	return filepath.Join(i.BaseDir, "manifest.json")
}

// ConfigFilePath returns the path to the `config.json` file.
func (i *Image) ConfigFilePath() string {
	return filepath.Join(i.BlobsDir(), "config.json")
}

// BlobsDir returns the path to layers should be written to.
func (i *Image) BlobsDir() string {
	return filepath.Join(i.BaseDir, "blobs")
}

// LayersDir returns the path to layers should be written to.
func (i *Image) LayersDir() string {
	return filepath.Join(i.BaseDir, "layers")
}

// Refresh reloads the missing image properties (from disk).
func (i *Image) Refresh() error {
	if err := i.loadConfig(); err != nil {
		return err
	}
	if err := i.loadManifest(); err != nil {
		return err
	}

	return nil
}

// ExposedPorts returns the list of exposed ports defined in the image.
func (i *Image) ExposedPorts() ([]network.ExposedPort, error) {
	return network.ParseExposedPorts(i.Config.Config.ExposedPorts)
}

// loadConfig loads the image config from disk.
func (i *Image) loadConfig() error {
	file := filepath.Join(i.ConfigFilePath())
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	i.Config = new(imagespec.Image)
	if err := json.Unmarshal(data, i.Config); err != nil {
		return err
	}

	return nil
}

// loadManifest loads the manifest data of an image from disk.
func (i *Image) loadManifest() error {
	data, err := os.ReadFile(i.ManifestFilePath())
	if err != nil {
		return err
	}

	i.Manifest = new(imagespec.Manifest)
	if err := json.Unmarshal(data, i.Manifest); err != nil {
		return err
	}

	return nil
}
