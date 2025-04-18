package clients

import (
	"time"

	"github.com/instruqt/jumppad/pkg/clients/command"
	"github.com/instruqt/jumppad/pkg/clients/connector"
	"github.com/instruqt/jumppad/pkg/clients/container"
	"github.com/instruqt/jumppad/pkg/clients/getter"
	"github.com/instruqt/jumppad/pkg/clients/helm"
	"github.com/instruqt/jumppad/pkg/clients/http"
	"github.com/instruqt/jumppad/pkg/clients/images"
	"github.com/instruqt/jumppad/pkg/clients/k8s"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/clients/nomad"
	"github.com/instruqt/jumppad/pkg/clients/system"
	"github.com/instruqt/jumppad/pkg/clients/tar"
	"github.com/instruqt/jumppad/pkg/utils"
)

type Clients struct {
	Docker         container.Docker
	ContainerTasks container.ContainerTasks
	Kubernetes     k8s.Kubernetes
	Helm           helm.Helm
	HTTP           http.HTTP
	Nomad          nomad.Nomad
	Command        command.Command
	Logger         logger.Logger
	Getter         getter.Getter
	System         system.System
	ImageLog       images.ImageLog
	Connector      connector.Connector
	TarGz          *tar.TarGz
}

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l logger.Logger) (*Clients, error) {
	dc, _ := container.NewDocker()

	kc := k8s.NewKubernetes(60*time.Second, l)

	hec := helm.NewHelm(l)

	ec := command.NewCommand(30*time.Second, l)

	hc := http.NewHTTP(1*time.Second, l)

	nc := nomad.NewNomad(hc, 1*time.Second, l)

	bp := getter.NewGetter(false)

	bc := &system.SystemImpl{}

	il := images.NewImageFileLog(utils.ImageCacheLog())

	tgz := &tar.TarGz{}

	ct, _ := container.NewDockerTasks(dc, il, tgz, l)

	co := connector.DefaultConnectorOptions()
	cc := connector.NewConnector(co)

	return &Clients{
		ContainerTasks: ct,
		Docker:         dc,
		Kubernetes:     kc,
		Helm:           hec,
		Command:        ec,
		HTTP:           hc,
		Nomad:          nc,
		Logger:         l,
		Getter:         bp,
		System:         bc,
		ImageLog:       il,
		Connector:      cc,
		TarGz:          tgz,
	}, nil
}
