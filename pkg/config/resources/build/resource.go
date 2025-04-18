package build

import (
	"fmt"
	"os"
	"path"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/pkg/config/resources/container"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/jumppad-labs/hclconfig/types"
)

// TypeBuild builds containers and other resources
const TypeBuild string = "build"

type Build struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Container BuildContainer `hcl:"container,block" json:"container"`

	// Outputs allow files or directories to be copied from the container
	Outputs []Output `hcl:"output,block" json:"outputs"`

	Registries []container.Image `hcl:"registry,block" json:"registries"` // Optional registry to push the image to

	// outputs

	// Image is the full local reference of the built image
	Image string `hcl:"image,optional" json:"image"`

	// Checksum is calculated from the Context files
	BuildChecksum string `hcl:"build_checksum,optional" json:"build_checksum,omitempty"`
}

type BuildContainer struct {
	DockerFile string            `hcl:"dockerfile,optional" json:"dockerfile,omitempty"` // Location of build file inside build context defaults to ./Dockerfile
	Context    string            `hcl:"context" json:"context"`                          // Path to build context
	Ignore     []string          `hcl:"ignore,optional" json:"ignore,omitempty"`         // Files to ignore in the build context, this is the same as .dockerignore
	Args       map[string]string `hcl:"args,optional" json:"args,omitempty"`             // Build args to pass  to the container
}

type Registry struct {
}

type Output struct {
	Source      string `hcl:"source" json:"source"`           // Source file or directory in container
	Destination string `hcl:"destination" json:"destination"` // Destination for copied file or directory
}

func (b *Build) Process() error {
	b.Container.Context = utils.EnsureAbsolute(b.Container.Context, b.Meta.File)

	// check that the Dockerfile exists inside the context folder
	// if not raise an error
	if b.Container.DockerFile != "" {
		path := path.Join(b.Container.Context, b.Container.DockerFile)
		_, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("the Dockerfile must located in the context folder, the file %s, does not exist in the context path %s", b.Container.DockerFile, b.Container.Context)
		}
	}

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(b.Meta.ID)
		if r != nil {
			kstate := r.(*Build)
			b.Image = kstate.Image

			// add the build checksum
			b.BuildChecksum = kstate.BuildChecksum
		}
	}

	return nil
}
