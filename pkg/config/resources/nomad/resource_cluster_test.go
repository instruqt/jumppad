package nomad

import (
	"os"
	"path"
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	ctypes "github.com/instruqt/jumppad/pkg/config/resources/container"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeNomadCluster, &NomadCluster{}, &ClusterProvider{})
}

func TestNomadClusterProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &NomadCluster{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},

		ServerConfig: "./server_config.hcl",
		ClientConfig: "./client_config.hcl",
		ConsulConfig: "./consul_config.hcl",

		Volumes: []ctypes.Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, path.Join(wd, "server_config.hcl"), c.ServerConfig)
	require.Equal(t, path.Join(wd, "client_config.hcl"), c.ClientConfig)
	require.Equal(t, path.Join(wd, "consul_config.hcl"), c.ConsulConfig)
	require.Equal(t, wd, c.Volumes[0].Source)
}

func TestNomadClusterProcessDoesNotSetAbsoluteForNonBindMounts(t *testing.T) {
	c := &NomadCluster{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},

		Volumes: []ctypes.Volume{
			{
				Type:        "volume",
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, "./", c.Volumes[0].Source)
}

func TestNomadClusterSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
  {
      "meta": {
        "id": "resource.nomad_cluster.test",
        "name": "test",
        "type": "nomad_cluster"
      },
      "api_port": 123,
      "connector_port": 124,
      "external_ip": "127.0.0.1",
      "server_container_name": "server.something.something",
      "client_container_name": ["1.client.something.something","2.client.something.something"],
      "config_dir": "abc/123"
  }
  ]
}`)

	c := &NomadCluster{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{ID: "resource.nomad_cluster.test"},
		},
	}

	c.Process()

	require.Equal(t, "127.0.0.1", c.ExternalIP)
	require.Equal(t, "server.something.something", c.ServerContainerName)
	require.Equal(t, []string{"1.client.something.something", "2.client.something.something"}, c.ClientContainerName)
	require.Equal(t, 123, c.APIPort)
	require.Equal(t, 124, c.ConnectorPort)
	require.Equal(t, "abc/123", c.ConfigDir)
}
