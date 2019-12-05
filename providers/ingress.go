package providers

import (
	"fmt"
	"github.com/shipyard-run/cli/clients"
	"github.com/shipyard-run/cli/config"
	"strconv"
)

type Ingress struct {
	config *config.Ingress
	client clients.Docker
}

func NewIngress(c *config.Ingress, cc clients.Docker) *Ingress {
	return &Ingress{c, cc}
}

func (i *Ingress) Create() error {
	// get the target ref
	t, ok := i.config.TargetRef.(*config.Container)
	if !ok {
		return fmt.Errorf("Only Container ingress is supported at present")
	}

	image := "shipyardrun/ingress:latest"
	command := make([]string, 0)

	// --network onprem docker.pkg.github.com/shipyard-run/ingress:latest --service-name consul.onprem.shipyard --port-remote 8500 --port-host 8500t
	// build the command based on the ports
	command = append(command, "--service-name")
	command = append(command, FQDN(t.Name, t.NetworkRef.Name))

	command = append(command, "--port-host")
	command = append(command, strconv.Itoa(i.config.Ports[0].Host))

	command = append(command, "--port-remote")
	command = append(command, strconv.Itoa(i.config.Ports[0].Remote))

	command = append(command, "--port-local")
	command = append(command, strconv.Itoa(i.config.Ports[0].Local))

	// ingress simply crease a container with specific options
	c := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.NetworkRef,
		Ports:      i.config.Ports,
		Image:      image,
		Command:    command,
	}

	p := NewContainer(c, i.client)

	return p.Create()
}

func (i *Ingress) Destroy() error {
	c := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.NetworkRef,
	}

	p := NewContainer(c, i.client)

	return p.Destroy()
}

func (i *Ingress) Lookup() (string, error) {
	return "", nil
}