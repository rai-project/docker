package docker

import (
	"context"
	"io"
	"runtime"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/command"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
)

// holdHijackedConnection handles copying input to and output from streams to the
// connection
func holdHijackedConnection(ctx context.Context, streams command.Streams, tty bool,
	inputStream io.ReadCloser, outputStream, errorStream io.Writer,
	resp types.HijackedResponse) error {
	var (
		err         error
		restoreOnce sync.Once
	)

	if inputStream != nil && tty {
		if err := setRawTerminal(streams); err != nil {
			return err
		}
		defer func() {
			restoreOnce.Do(func() {
				restoreTerminal(streams, inputStream)
			})
		}()
	}

	receiveStdout := make(chan error, 1)
	if outputStream != nil || errorStream != nil {
		go func() {
			// When TTY is ON, use regular copy
			if tty && outputStream != nil {
				_, err = io.Copy(outputStream, resp.Reader)
				// we should restore the terminal as soon as possible once connection end
				// so any following print messages will be in normal type.
				if inputStream != nil {
					restoreOnce.Do(func() {
						restoreTerminal(streams, inputStream)
					})
				}
			} else {
				_, err = stdcopy.StdCopy(outputStream, errorStream, resp.Reader)
			}

			receiveStdout <- err
		}()
	}

	stdinDone := make(chan struct{})
	go func() {
		if inputStream != nil {
			io.Copy(resp.Conn, inputStream)
			// we should restore the terminal as soon as possible once connection end
			// so any following print messages will be in normal type.
			if tty {
				restoreOnce.Do(func() {
					restoreTerminal(streams, inputStream)
				})
			}
		}
		if err := resp.CloseWrite(); err != nil {
			log.Debugf("Couldn't send EOF: %s", err)
		}
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		if err != nil {
			log.Debugf("Error receiveStdout: %s", err)
			return errors.Wrap(err, "while hijacking stdout")
		}
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			select {
			case err := <-receiveStdout:
				if err != nil {
					log.Debugf("Error receiveStdout: %s", err)
					return errors.Wrap(err, "stdin done while hijacking stdout")
				}
			case <-ctx.Done():
			}
		}
	case <-ctx.Done():
	}

	return nil
}

func setRawTerminal(streams command.Streams) error {
	if err := streams.In().SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out().SetRawTerminal()
}

func restoreTerminal(streams command.Streams, in io.Closer) error {
	streams.In().RestoreTerminal()
	streams.Out().RestoreTerminal()
	// WARNING: DO NOT REMOVE THE OS CHECK !!!
	// For some reason this Close call blocks on darwin..
	// As the client exists right after, simply discard the close
	// until we find a better solution.
	if in != nil && runtime.GOOS != "darwin" {
		return in.Close()
	}
	return nil
}

type stream struct {
	stdin  io.ReadCloser
	stdout io.Writer
	stderr io.Writer
}

func (s *stream) In() *command.InStream {
	if s.stdin == nil {
		return nil
	}
	return command.NewInStream(s.stdin)
}

func (s *stream) Out() *command.OutStream {
	if s.stdout == nil {
		return nil
	}
	return command.NewOutStream(s.stdout)
}

func (s *stream) Err() io.Writer {
	return s.stderr
}
