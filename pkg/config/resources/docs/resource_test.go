package docs

import (
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeDocs, &Docs{}, &DocsProvider{})
}

func TestDocsProcessSetsAbsolute(t *testing.T) {
	h := &Docs{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
	}

	err := h.Process()
	require.NoError(t, err)
}

func TestDocsLoadsValuesFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"meta": {
				"id": "resource.docs.test",
  	    "name": "test",
  	    "type": "docs"
			},
			"fqdn": "fqdn.mine"
	}
	]
}`)

	docs := &Docs{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				File: "./",
				ID:   "resource.docs.test",
			},
		},
	}

	err := docs.Process()
	require.NoError(t, err)

	require.Equal(t, "fqdn.mine", docs.ContainerName)
}
