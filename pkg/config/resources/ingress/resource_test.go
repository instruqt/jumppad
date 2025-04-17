package ingress

import (
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeIngress, &Ingress{}, &Provider{})
}

func TestIngressSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"meta": {
				"id": "resource.ingress.test",
      	"name": "test",
      	"type": "ingress"
			},
			"ingress_id": "42",
			"local_address": "127.0.0.1"
	}
	]
}`)

	c := &Ingress{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				ID: "resource.ingress.test",
			},
		},
	}

	c.Process()

	require.Equal(t, "42", c.IngressID)
	require.Equal(t, "127.0.0.1", c.LocalAddress)
}
