package container

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/clients/tar"
	"github.com/instruqt/jumppad/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateVolumeDoesNothingWhenVolumeExists(t *testing.T) {
	_, md, mic := createContainerConfig()

	testutils.RemoveOn(&md.Mock, "VolumeList")

	args := volume.ListOptions{Filters: filters.NewArgs()}
	args.Filters.Add("name", "test.volume.jmpd.in")
	md.On("VolumeList", mock.Anything, args).Return(volume.ListResponse{Volumes: []*volume.Volume{{}}}, nil)

	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))
	_, err := p.CreateVolume("test")
	assert.NoError(t, err)

	md.AssertNotCalled(t, "VolumeCreate")
}

func TestCreateVolumeReturnsErrorWhenVolumeListError(t *testing.T) {
	_, md, mic := createContainerConfig()
	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	testutils.RemoveOn(&md.Mock, "VolumeList")

	args := volume.ListOptions{Filters: filters.NewArgs()}
	args.Filters.Add("name", "test.volume.jmpd.in")
	md.On("VolumeList", mock.Anything, args).Return(volume.ListResponse{}, fmt.Errorf("Boom"))

	_, err := p.CreateVolume("test")
	assert.Error(t, err)

	md.AssertNotCalled(t, "VolumeCreate")
}

func TestCreateVolumeCreatesSuccesfully(t *testing.T) {
	_, md, mic := createContainerConfig()
	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	id, err := p.CreateVolume("test")
	assert.NoError(t, err)

	md.AssertCalled(t, "VolumeCreate", mock.Anything, mock.Anything)
	assert.Equal(t, "test_volume", id)
}

func TestRemoveVolumeRemotesSuccesfully(t *testing.T) {
	_, md, mic := createContainerConfig()
	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	err := p.RemoveVolume("test")
	assert.NoError(t, err)

	md.AssertCalled(t, "VolumeRemove", mock.Anything, "test.volume.jmpd.in", true)
}
