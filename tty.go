package docker

import (
	"os"
	gosignal "os/signal"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/cli/command"
	"github.com/moby/moby/pkg/signal"
	"github.com/moby/moby/pkg/term"
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
	stdin := c.client.options.stdin
	if isExec && e.Stdin != nil {
		if s, ok := e.Stdin.(*command.InStream); ok {
			stdin = s
		}
	}
	if stdin == nil {
		return nil
	}

	if !stdin.IsTerminal() {
		log.Debug("unable to monitor tty size, because stdin is not a terminal")
		return errors.New("not a terminal")
	}

	getTtySize := func() (uint, uint) {
		ws, err := term.GetWinsize(stdin.FD())
		if err != nil {
			return 0, 0
		}
		return uint(ws.Height), uint(ws.Width)
	}
	doResizeTty := func() {
		h, w := getTtySize()
		resizeTty(c, e, h, w, isExec)
	}

	doResizeTty()

	if runtime.GOOS == "windows" {
		go func() {
			prevH, prevW := getTtySize()
			for {
				time.Sleep(time.Millisecond * 250)
				h, w := getTtySize()
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
