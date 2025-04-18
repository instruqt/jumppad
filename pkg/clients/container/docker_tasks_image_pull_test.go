package container

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/system"
	"github.com/instruqt/jumppad/pkg/clients/container/mocks"
	dtypes "github.com/instruqt/jumppad/pkg/clients/container/types"
	imocks "github.com/instruqt/jumppad/pkg/clients/images/mocks"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/clients/tar"
	"github.com/instruqt/jumppad/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupImagePullMocks() (*mocks.Docker, *imocks.ImageLog) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		io.NopCloser(strings.NewReader("hello world")),
		nil,
	)

	mic := &imocks.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)
	mic.On("Read", mock.Anything, mock.Anything).Return([]string{}, nil)

	return md, mic
}

func createImagePullConfig() (dtypes.Image, *mocks.Docker, *imocks.ImageLog) {
	ic := dtypes.Image{
		Name: "consul:1.6.1",
	}

	mk, mic := setupImagePullMocks()
	return ic, mk, mic
}

func setupImagePull(t *testing.T, cc dtypes.Image, md *mocks.Docker, mic *imocks.ImageLog, force bool) {
	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	// create the container
	err := p.PullImage(cc, force)
	assert.NoError(t, err)
}

func TestPullImageWhenNOTCached(t *testing.T) {
	cc, md, mic := createImagePullConfig()
	setupImagePull(t, cc, md, mic, false)

	// test calls list image with a canonical image reference
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: cc.Name})
	md.AssertCalled(t, "ImageList", mock.Anything, image.ListOptions{Filters: args})

	// test pulls image replacing the short name with the canonical registry name
	md.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical(cc.Name), image.PullOptions{})

	// test adds to the cache log
	mic.AssertCalled(t, "Log", mock.Anything, mock.Anything)
}

func TestPullImageWithCredentialsWhenNOTCached(t *testing.T) {
	cc, md, mic := createImagePullConfig()
	cc.Username = "nicjackson"
	cc.Password = "S3cur1t11"

	setupImagePull(t, cc, md, mic, false)

	// test calls list image with a canonical image reference
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: cc.Name})
	md.AssertCalled(t, "ImageList", mock.Anything, image.ListOptions{Filters: args})

	// test pulls image replacing the short name with the canonical registry name
	// adding credentials to image pull
	ipo := image.PullOptions{RegistryAuth: createRegistryAuth(cc.Username, cc.Password)}
	md.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical(cc.Name), ipo)

}

func TestPullImageWithValidCredentials(t *testing.T) {
	cc, md, mic := createImagePullConfig()
	cc.Username = "nicjackson"
	cc.Password = "S3cur1t11"

	setupImagePull(t, cc, md, mic, false)

	ipo := testutils.GetCalls(&md.Mock, "ImagePull")[0].Arguments[2].(image.PullOptions)

	d, err := base64.StdEncoding.DecodeString(ipo.RegistryAuth)
	assert.NoError(t, err)
	assert.Equal(t, `{"username":"nicjackson","password":"S3cur1t11"}`, string(d))
}

func TestDoNotPullImageWhenLocalImage(t *testing.T) {
	cc, md, mic := createImagePullConfig()
	cc.Name = "jumppad.dev/localcache/mine:latest"
	setupImagePull(t, cc, md, mic, false)

	md.AssertNotCalled(t, "ImagePull", mock.Anything, mock.Anything, mock.Anything)
	mic.AssertNotCalled(t, "Log", mock.Anything, mock.Anything)
}

func TestDoNOtPullImageWhenCached(t *testing.T) {
	cc, md, mic := createImagePullConfig()

	// remove the default image list which returns 0 cached images
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything).Return([]image.Summary{{ID: "abc"}}, nil)

	setupImagePull(t, cc, md, mic, false)

	md.AssertNotCalled(t, "ImagePull", mock.Anything, mock.Anything, mock.Anything)
	mic.AssertNotCalled(t, "Log", mock.Anything, mock.Anything)
}

func TestPullImageAlwaysWhenForce(t *testing.T) {
	cc, md, mic := createImagePullConfig()

	setupImagePull(t, cc, md, mic, true)

	md.AssertNotCalled(t, "ImageList", mock.Anything, mock.Anything)
	md.AssertCalled(t, "ImagePull", mock.Anything, mock.Anything, mock.Anything)
	mic.AssertCalled(t, "Log", mock.Anything, mock.Anything)
}
