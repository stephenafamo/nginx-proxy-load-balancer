package workers

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/stephenafamo/janus/monitor"
	"github.com/stephenafamo/warden/internal"
)

type NginxServer struct {
	Settings internal.Settings
	Monitor  monitor.Monitor
}

func (n NginxServer) Play(ctx context.Context) error {
	if n.Settings.TESTING {
		return n.dev(ctx)
	}

	return n.prod(ctx)
}

func (n NginxServer) dev(ctx context.Context) error {
	// Just wait
	<-ctx.Done()
	return nil
}

func (n NginxServer) prod(ctx context.Context) error {

	cmd := exec.Command("nginx", "-g", "daemon off;")

	go func() {
		<-ctx.Done()

		if cmd.Process.Pid < 1 {
			// Process has not started or has already exited
			return
		}

		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			err = fmt.Errorf("error sending Interrupt signal to NGINX: %w", err)
			n.Monitor.CaptureException(err)
		}
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"Can't start NGINX: %s: %s",
			err,
			output,
		)
	}

	return nil
}
