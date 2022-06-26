package registry

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/artyom/untar"
	"github.com/opencontainers/go-digest"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yaman/image"
)

type registryOpts struct {
	AuthURL      string
	Service      string
	IndexBaseURL string
}

type token struct {
	Token       string    `json:"token"`
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	IssuedAt    time.Time `json:"issued_at"`
}

type registryClient struct {
	httpClient

	BaseURL string
}

func (c registryClient) GetManifest(img *image.Image) (*imagespec.Manifest, error) {
	resp, err := c.Head(
		fmt.Sprintf("%s/%s/manifests/%s", c.BaseURL, img.Name, img.Version),
		map[string]string{"Accept": "application/vnd.docker.distribution.manifest.v2+json"},
	)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	manifestDigest := resp.Header.Get("docker-content-digest")
	if manifestDigest == "" {
		return nil, errors.New("missing manifest digest")
	}

	switch manifestType := resp.Header.Get("content-type"); manifestType {
	case "application/vnd.docker.distribution.manifest.v2+json":
		break

	default:
		return nil, fmt.Errorf("unsupported manifest type: %s", manifestType)
	}

	resp, err = c.Get(
		fmt.Sprintf("%s/%s/manifests/%s", c.BaseURL, img.Name, manifestDigest),
		map[string]string{"Accept": "application/vnd.docker.distribution.manifest.v2+json"},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	manifest := new(imagespec.Manifest)
	if err := json.NewDecoder(resp.Body).Decode(manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

func (c registryClient) GetImage(img *image.Image, manifest *imagespec.Manifest) (*imagespec.Image, error) {
	resp, err := c.Get(
		fmt.Sprintf("%s/%s/blobs/%s", c.BaseURL, img.Name, manifest.Config.Digest),
		map[string]string{"Accept": manifest.Config.MediaType},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	config := new(imagespec.Image)
	if err := json.NewDecoder(resp.Body).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (c registryClient) DownloadAndUnpackLayer(img *image.Image, layer imagespec.Descriptor, diffID digest.Digest) error {
	resp, err := c.Get(
		fmt.Sprintf("%s/%s/blobs/%s", c.BaseURL, img.Name, layer.Digest),
		map[string]string{"Accept": layer.MediaType},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch layer.MediaType {
	case "application/vnd.docker.image.rootfs.diff.tar.gzip":
		r, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer r.Close()

		layerDir := filepath.Join(img.LayersDir(), diffID.Hex())
		if err := untar.Untar(r, layerDir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported layer type '%s'", layer.MediaType)
	}

	logrus.WithFields(logrus.Fields{
		"image":  img.FQIN(),
		"digest": layer.Digest,
	}).Debug("unpacked layer")

	return nil
}

func PullFromRegistry(img *image.Image, pullOpts PullOpts, opts registryOpts) error {
	logger := logrus.WithField("image", img.FQIN())

	if pullOpts.Output != nil {
		fmt.Fprintf(pullOpts.Output, "Pulling %s\n", img.FQIN())
	}

	if err := os.MkdirAll(img.BlobsDir(), 0o755); err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s?service=%s&scope=repository:%s:pull",
		opts.AuthURL,
		opts.Service,
		img.Name,
	)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	var t token
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return err
	}
	resp.Body.Close()
	logger.Debug("got authentication token")

	c := registryClient{newHttpClientWithAuthToken(t.Token), opts.IndexBaseURL}

	manifest, err := c.GetManifest(img)
	if err != nil {
		return err
	}
	manifestFile, err := os.Create(img.ManifestFilePath())
	if err != nil {
		return err
	}
	if err := json.NewEncoder(manifestFile).Encode(manifest); err != nil {
		return err
	}
	logger.Debug("wrote manifest.json")

	config, err := c.GetImage(img, manifest)
	if err != nil {
		return err
	}
	configFile, err := os.Create(img.ConfigFilePath())
	if err != nil {
		return err
	}
	if err := json.NewEncoder(configFile).Encode(config); err != nil {
		return err
	}
	logger.Debug("wrote blobs/config.json")

	for index, layer := range manifest.Layers {
		diffID := config.RootFS.DiffIDs[index]

		if err := c.DownloadAndUnpackLayer(img, layer, diffID); err != nil {
			return err
		}
	}

	if pullOpts.Output != nil {
		fmt.Fprintf(pullOpts.Output, "Digest: %s\n", manifest.Config.Digest.String())
	}

	return nil
}
