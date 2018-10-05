package docker

import (
	"context"
	"io"
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
)

// redirectResponseToOutputStream redirect the response stream to stdout and stderr. When tty is true, all stream will
// only be redirected to stdout.
func redirectResponseToOutputStream(tty bool, outputStream, errorStream io.Writer, resp io.Reader) error {
	if outputStream == nil {
		outputStream = ioutil.Discard
	}
	if errorStream == nil {
		errorStream = ioutil.Discard
	}
	var err error
	if tty {
		_, err = io.Copy(outputStream, resp)
	} else {
		_, err = stdcopy.StdCopy(outputStream, errorStream, resp)
	}
	return err
}

func holdHijackedConnection(ctx context.Context, streams Streams, tty bool,
	inputStream io.ReadCloser, outputStream, errorStream io.Writer,
	resp types.HijackedResponse) error {
	var (
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

	receiveStdout := make(chan error)
	if outputStream != nil || errorStream != nil {
		go func() {
			receiveStdout <- redirectResponseToOutputStream(tty, outputStream, errorStream, resp.Reader)
		}()
	}

	stdinDone := make(chan struct{})
	go func() {
		if inputStream != nil {
			io.Copy(resp.Conn, inputStream)
		}
		restoreOnce.Do(func() {
			restoreTerminal(streams, inputStream)
		})
		resp.CloseWrite()
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		if err != nil {
			log.Debugf("Error receiveStdout: %s", err)
			return errors.Wrap(err, "while hijacking stdout")
		}
		return err
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			return <-receiveStdout
		}
	case <-ctx.Done():
		break
	}
	return nil
}

// holdHijackedConnection handles copying input to and output from streams to the
// connection
func holdHijackedConnection1(ctx context.Context, streams Streams, tty bool,
	inputStream io.ReadCloser, outputStream, errorStream io.Writer,
	resp types.HijackedResponse) error {
	var (
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
			var err error
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
		return nil
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			select {
			case err := <-receiveStdout:
				if err != nil {
					log.Debugf("Error receiveStdout: %s", err)
					return errors.Wrap(err, "stdin done while hijacking stdout")
				}
				return nil
			case <-ctx.Done():
				break
			}
		}
	case <-ctx.Done():
		break
	}

	return nil
}

func setRawTerminal(streams Streams) error {
	if err := streams.In().SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out().SetRawTerminal()
}

func restoreTerminal(streams Streams, in io.Closer) error {
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
