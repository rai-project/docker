package docker

import (
	"os"
	gosignal "os/signal"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/command"
	"github.com/docker/docker/pkg/signal"
	"github.com/pkg/errors"
)

func resizeTty(c *Container, e *Execution, height, width uint, isExec bool) {
	if height == 0 && width == 0 {
		return
	}

	options := types.ResizeOptions{
		Height: height,
		Width:  width,
	}

	client := c.client
	ctx := c.options.context

	var err error
	if isExec {
		err = client.ContainerExecResize(ctx, e.execID, options)
	} else {
		err = client.ContainerResize(ctx, c.ID, options)
	}

	if err != nil {
		logrus.WithError(err).Debug("resize error")
	}
}

func monitorTtySize(c *Container, e *Execution, isExec bool) error {
	stdout := c.client.options.stdout
	if isExec && e.Stdout != nil {
		if s, ok := e.Stdout.(*command.OutStream); ok {
			stdout = s
		}
	}
	if !stdout.IsTerminal() {
		log.Debug("unable to monitor tty size, because stdout is not a terminal")
		return errors.New("not a terminal")
	}

	doResizeTty := func() {
		h, w := stdout.GetTtySize()
		resizeTty(c, e, h, w, isExec)
	}

	doResizeTty()

	if runtime.GOOS == "windows" {
		go func() {
			prevH, prevW := stdout.GetTtySize()
			for {
				time.Sleep(time.Millisecond * 250)
				h, w := stdout.GetTtySize()
				if h != prevH || w != prevW {
					doResizeTty()
				}
				prevH, prevW = h, w
			}
		}()
	} else {
		sigchan := make(chan os.Signal, 1)
		gosignal.Notify(sigchan, signal.SIGWINCH)
		go func() {
			for range sigchan {
				doResizeTty()
			}
		}()
	}
	return nil
}

func (c *Container) MonitorTtySize() error {
	return monitorTtySize(c, nil, false)
}

func (e *Execution) MonitorTtySize() error {
	return monitorTtySize(e.container, e, true)
}

func (c *Container) ForwardAllSignals() chan os.Signal {
	sigc := make(chan os.Signal, 128)
	signal.CatchAll(sigc)
	go func() {
		for s := range sigc {
			if s == signal.SIGCHLD || s == signal.SIGPIPE {
				continue
			}
			var sig string
			for sigStr, sigN := range signal.SignalMap {
				if sigN == s {
					sig = sigStr
					break
				}
			}
			if sig == "" {
				log.Errorf("Unsupported signal: %v. Discarding.\n", s)
				continue
			}

			if err := c.killWithSignal(sig); err != nil {
				logrus.WithError(err).Debug("sending signal")
			}
		}
	}()
	return sigc
}
