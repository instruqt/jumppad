package container

import (
	"io"

	"github.com/instruqt/jumppad/pkg/clients/container/types"
)

// ContainerTasks is a task oriented client which abstracts
// the underlying container technology from the providers
// this allows different concrete implementations such as Docker, or ContainerD
// without needing to change the provider code.
//
// The Docker SDK can also be quite terse, the API design for this client
// is design is centered around performing a task such as CreateContainer,
// this may be composed of many individual SDK calls.
//
//go:generate mockery --name ContainerTasks --filename container_tasks.go
type ContainerTasks interface {
	SetForce(bool)
	// CreateContainer creates a new container for the given configuration
	// if successful CreateContainer returns the ID of the created container and a nil error
	// if not successful CreateContainer returns a blank string for the id and an error message
	CreateContainer(*types.Container) (id string, err error)
	// Container Info returns an anonymous interface corresponding to the container info
	// returns error when unable to read info such as when the container does not exist.
	ContainerInfo(id string) (interface{}, error)
	// RemoveContainer stops and removes a running container
	RemoveContainer(id string, force bool) error
	// BuildContainer builds a container based on the given configuration
	// If a cached image already exists Build will noop
	// When force is specified BuildContainer will rebuild the container regardless of cached images
	// Returns the canonical name of the built image and an error
	BuildContainer(config *types.Build, force bool) (string, error)
	// CreateVolume creates a new volume with the given name.
	// If successful the id of the newly created volume is returned
	CreateVolume(name string) (id string, err error)
	// RemoveVolume removes a volume with the given name
	RemoveVolume(name string) error
	// FindImageInLocalRegistry returns the unique identifier for an image specified by the given
	// tag in the local registry. If no image is found the function returns an
	// empty id and no error
	// An error is only returned on internal errors when communicating with the
	// Docker API
	FindImageInLocalRegistry(image types.Image) (string, error)
	// FindImageInLocalRegistry returns the unique identifiers for an image specified by the given
	// tag in the local registry. If no image is found the function returns an
	// empty id and no error
	// An error is only returned on internal errors when communicating with the
	// Docker API
	FindImagesInLocalRegistry(filter string) ([]string, error)
	// PullImage pulls a Docker image from the registry if it is not already
	// present in the local cache.
	// If the Username and Password config options are set then PullImage will attempt to
	// authenticate with the registry before pulling the image.
	// If the force parameter is set then PullImage will pull regardless of the image already
	// being cached locally.
	PullImage(image types.Image, force bool) error
	// PushImage pushes an image to the registry
	PushImage(image types.Image) error
	// FindContainerIDs returns the Container IDs for the given container name
	FindContainerIDs(containerName string) ([]string, error)
	// RemoveImage removes the image with the given id from the local registry
	RemoveImage(id string) error
	// ContainerLogs attaches to the container and streams the logs to the returned
	// io.ReadCloser.
	// Returns an error if the container is not running
	ContainerLogs(id string, stdOut, stdErr bool) (io.ReadCloser, error)
	// CopyFromContainer allows the copying of a file from a container
	CopyFromContainer(id, src, dst string) error
	// CopyToContainer allows a file to be copied into a container
	CopyFileToContainer(id, src, dst string) error
	// CreateFileInContainer creates a file with the given contents and name in the container containerID and
	// stores it in the container at the directory path.
	CreateFileInContainer(containerID, contents, filename, path string) error
	// CopyLocaDockerImageToVolume copies the docker images to the docker volume as a
	// compressed archive.
	// the path in the docker volume where the archive is created is returned
	// along with any errors.
	CopyLocalDockerImagesToVolume(images []string, volume string, force bool) ([]string, error)

	//CopyFilesToVolume copies the files to the path in a Docker volume
	CopyFilesToVolume(volume string, files []string, path string, force bool) ([]string, error)
	// Execute command allows the execution of commands in a running docker container
	// id is the id of the container to execute the command in
	// command is a slice of strings to execute
	// timeout in seconds
	// writer [optional] will be used to write any output from the command execution.
	ExecuteCommand(id string, command []string, env []string, workingDirectory string, user, group string, timeout int, writer io.Writer) (int, error)
	// ExecuteScript allows the execution of a script in a running docker container
	// id is the id of the container to execute the command in
	// contents is the contents of the script to execute
	// writer [optional] will be used to write any output from the command execution.
	ExecuteScript(id string, contents string, env []string, workingDirectory string, user, group string, timeout int, writer io.Writer) (int, error)
	// AttachNetwork attaches a container to a network
	// if aliases is set an alias for the container name will be added
	// if ipAddress is not null then a user defined ipaddress will be used
	AttachNetwork(network, containerid string, aliases []string, ipaddress string) error
	// DetatchNetwork disconnects a container from the network
	DetachNetwork(network, containerid string) error
	// ListNetworks lists the networks a container is attached to
	ListNetworks(id string) []types.NetworkAttachment
	// FindNetwork returns a network using the unique resource id
	FindNetwork(id string) (types.NetworkAttachment, error)

	// CreateShell in the running container and attach
	CreateShell(id string, command []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error

	// TagImage tags an image with the given tag
	TagImage(source, destination string) error

	// Returns basic information related to the Docker Engine
	EngineInfo() *types.EngineInfo
}
