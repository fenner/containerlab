package utils

import (
	"strings"
	"time"

	"github.com/scrapli/scrapligo/driver/base"
	"github.com/scrapli/scrapligo/driver/core"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/transport"
	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/srlinux-scrapli"
)

var (
	// map of commands per platform which start a CLI app
	NetworkOSCLICmd = map[string]string{
		"arista_eos":    "Cli",
		"nokia_srlinux": "sr_cli",
	}

	// map of the cli exec command and its argument per runtime
	// which is used to spawn CLI session
	CLIExecCommand = map[string]map[string]string{
		"docker": {
			"exec": "docker",
			"open": "exec -it",
		},
		"containerd": {
			"exec": "ctr",
			"open": "-n clab task exec -t --exec-id clab",
		},
	}
)

// SpawnCLIviaExec spawns a CLI session over container runtime exec function
// end ensures the CLI is available to be used for sending commands over
func SpawnCLIviaExec(platform, contName, runtime string) (*network.Driver, error) {
	var d *network.Driver
	var err error

	switch platform {
	case "nokia_srlinux":
		d, err = srlinux.NewSRLinuxDriver(
			contName,
			base.WithAuthBypass(true),
			// disable transport timeout
			base.WithTimeoutTransport(0),
		)
		// jack up PtyWidth, since we use `docker exec` to enter certificate and key strings
		// and these are lengthy
		d.Transport.BaseTransportArgs.PtyWidth = 5000
	default:
		d, err = core.NewCoreDriver(
			contName,
			platform,
			base.WithAuthBypass(true),
			base.WithTimeoutTransport(0),
		)
	}

	if err != nil {
		log.Errorf("failed to create driver for device %s; error: %+v\n", err, contName)
		return nil, err
	}

	execCmd := CLIExecCommand[runtime]["exec"]
	openCmd := strings.Split(CLIExecCommand[runtime]["open"], " ")

	t, _ := d.Transport.Impl.(transport.SystemTransport)
	t.SetExecCmd(execCmd)
	t.SetOpenCmd(append(openCmd, contName, NetworkOSCLICmd[platform]))

	transportReady := false
	for !transportReady {
		if err := d.Open(); err != nil {
			log.Debugf("%s - Cli not ready (%s) - waiting.", contName, err)
			time.Sleep(time.Second * 2)
			continue
		}
		transportReady = true
		log.Debugf("%s - Cli ready.", contName)
	}

	return d, err
}
