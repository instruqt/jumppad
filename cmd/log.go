package cmd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/fatih/color"
	hcltypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/spf13/cobra"

	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/cache"
	ct "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

func newLogCmd(dc container.Docker, stdout, stderr io.Writer) *cobra.Command {
	logCmd := &cobra.Command{
		Use:     "logs [resource]",
		Short:   "Tails logs for running jumppad resources",
		Long:    "Tails logs for running jumppad resources",
		Aliases: []string{"log"},
		Example: `
  # Tail logs for all running resources
	jumppad logs

	# Tail logs for a specific resource
	jumppad logs resource.container.nginx
	`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: getResources,
		RunE:              newLogCmdFunc(dc, stdout, stderr),
	}

	return logCmd
}

var termColors = []color.Attribute{
	color.FgRed,
	color.FgGreen,
	color.FgYellow,
	color.FgBlue,
	color.FgMagenta,
	color.FgCyan,
	color.FgWhite,
}

func getResources(cmd *cobra.Command, args []string, complete string) ([]string, cobra.ShellCompDirective) {
	loggable, err := getLoggable()
	if err != nil {
		return []string{err.Error()}, cobra.ShellCompDirectiveNoFileComp
	}

	return loggable, cobra.ShellCompDirectiveNoFileComp
}

func newLogCmdFunc(dc container.Docker, stdout, stderr io.Writer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := createLogger()
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		waitGroup := sync.WaitGroup{}

		var loggable []string

		if len(args) == 1 {
			cfg, err := config.LoadState()
			if err != nil {
				return errors.New("unable to read state file")
			}

			r, err := cfg.FindResource(args[0])
			if err != nil {
				return fmt.Errorf("%s not found: %s", args[0], err)
			}

			loggable = getFQDNForResource(r)
		} else {
			var err error
			loggable, err = getLoggable()
			if err != nil {
				return err
			}
		}

		ctx := context.Background()

		for _, r := range loggable {
			rc, err := dc.ContainerLogs(
				ctx,
				r,
				dcontainer.LogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Follow:     true,
					Tail:       "40",
				},
			)

			if err == nil {
				waitGroup.Add(1)
				go func(rc io.ReadCloser, name string, c color.Attribute, log logger.Logger) {
					writeLogOutput(rc, stdout, stderr, name, c, log)
					waitGroup.Done()
				}(rc, r, getRandomColor(), log)
			} else {
				log.Error("Unable to get logs for container", "error", err)
			}
		}

		// send an interrupt when the waitGroup is done
		go func() {
			waitGroup.Wait()
			log.Info("No more logs to tail")
			sigs <- os.Interrupt
		}()

		// block until a signal is received
		<-sigs

		return nil
	}
}

// if this methods returns and error, it will get returned as shell-completion data
// otherwise fmt.println() gets lost
func getLoggable() ([]string, error) {
	cfg, err := config.LoadState()
	if err != nil {
		return nil, errors.New("unable to read state file")
	}

	loggable := []string{}
	for _, r := range cfg.Resources {
		if r.GetDisabled() {
			continue
		}

		loggable = append(loggable, getFQDNForResource(r)...)
	}
	return loggable, nil
}

func getFQDNForResource(r hcltypes.Resource) []string {
	fqdns := []string{}

	switch r.Metadata().Type {
	case ct.TypeContainer:
		fqdns = append(fqdns, utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type))
	case k8s.TypeK8sCluster:
		fqdns = append(fqdns, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type)))
	case nomad.TypeNomadCluster:
		fqdns = append(fqdns, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type)))

		// add the client nodes
		nomad := r.(*nomad.NomadCluster)
		for n := 0; n < nomad.ClientNodes; n++ {
			fqdns = append(fqdns, fmt.Sprintf("%d.%s.%s", n+1, "client", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type)))
		}
	case ct.TypeSidecar:
		fallthrough
	case cache.TypeImageCache:
		fqdns = append(fqdns, utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type))
	}

	return fqdns
}

func getRandomColor() color.Attribute {
	return termColors[rand.Intn(len(termColors)-1)]
}

func writeLogOutput(rc io.ReadCloser, stdout, stderr io.Writer, name string, c color.Attribute, log logger.Logger) {
	hdr := make([]byte, 8)
	colorWriter := color.New(c)

	for {
		_, err := rc.Read(hdr)
		if err != nil {
			log.Error("Unable to read from log stream", "name", name, "error", err)
			return
		}

		var w io.Writer
		switch hdr[0] {
		case 1:
			w = stdout
		default:
			w = stderr
		}

		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = rc.Read(dat)
		if err != nil {
			log.Error("Unable to read from log stream", "name", name, "error", err)
			return
		}

		name = strings.TrimSuffix(name, utils.LocalTLD)
		colorWriter.Fprintf(w, "[%s]   %s", name, string(dat))
	}
}
