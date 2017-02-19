package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http/httputil"
	"runtime"
	"strings"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/command"
	"github.com/docker/docker/pkg/promise"
	"github.com/docker/docker/pkg/stdcopy"
	shellwords "github.com/junegunn/go-shellwords"
	"github.com/pkg/errors"
)

type Execution struct {
	container  *Container
	context    context.Context
	cancelFunc context.CancelFunc

	// Path is the path or name of the command in the container.
	Path string

	// Arguments to the command in the container, excluding the command
	// name as the first argument.
	Args []string

	// Env is environment variables to the command. If Env is nil, Run will use
	// Env specified on Method or pre-built container image.
	Env []string

	// Dir specifies the working directory of the command. If Dir is the empty
	// string, Run uses Dir specified on Method or pre-built container image.
	Dir string

	// Stdin specifies the process's standard input.
	// If Stdin is nil, the process reads from the null device (os.DevNull).
	//
	// Run will not close the underlying handle if the Reader is an *os.File
	// differently than os/exec.
	Stdin io.ReadCloser

	// Stdout and Stderr specify the process's standard output and error.
	// If either is nil, they will be redirected to the null device (os.DevNull).
	//
	// Run will not close the underlying handles if they are *os.File differently
	// than os/exec.
	Stdout io.Writer
	Stderr io.Writer

	isStarted      bool
	execID         string
	wc             chan error
	closeAfterWait []io.Closer
}

func NewExecution(container *Container, args ...string) (*Execution, error) {
	ctx, cancelFunc := context.WithCancel(container.options.context)
	return &Execution{
		container:  container,
		Path:       args[0],
		Args:       args[1:],
		context:    ctx,
		cancelFunc: cancelFunc,
		isStarted:  false,
	}, nil
}

func NewExecutionFromString(container *Container, shell string) (*Execution, error) {
	args, err := shellwords.Parse(shell)
	if err != nil {
		log.WithError(err).WithField("cmd", shell).Error("Failed to parse command line")
		return nil, errors.Wrapf(err, "Docker execution:: failed to parse command line %v", shell)
	}
	return NewExecution(container, args...)
}

func (e *Execution) CombinedOutput() ([]byte, error) {
	if e.Stdout != nil {
		return nil, errors.New("Docker execution:: Stdout already set")
	}
	if e.Stderr != nil {
		return nil, errors.New("Docker execution:: Stderr already set")
	}
	var b bytes.Buffer
	e.Stdout, e.Stderr = &b, &b
	err := e.Run()
	return b.Bytes(), err
}

func (e *Execution) Output() ([]byte, error) {
	if e.Stdout != nil {
		return nil, errors.New("Docker execution: Stdout already set")
	}
	var stdout, stderr bytes.Buffer
	e.Stdout = &stdout

	captureErr := e.Stderr == nil
	if captureErr {
		e.Stderr = &stderr
	}
	err := e.Run()
	if err != nil && captureErr {
		if ee, ok := err.(*ExitError); ok {
			ee.Stderr = stderr.Bytes()
		}
	}
	return stdout.Bytes(), err
}

func (e *Execution) Run() error {
	if err := e.Start(); err != nil {
		return err
	}
	return e.Wait()
}

func (e *Execution) Start() error {

	container := e.container
	client := container.client

	if e.Stdin == nil {
		e.Stdin = client.options.stdin
	}
	if e.Stdout == nil {
		e.Stdout = client.options.stdout
	}
	if e.Stderr == nil {
		e.Stderr = client.options.stderr
	}
	isTty := container.options.containerConfig.Tty

	cmd := append([]string{e.Path}, e.Args...)
	execOpts := types.ExecConfig{
		AttachStdin:  e.Stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          isTty,
		Cmd:          cmd,
		Detach:       true,
		User:         container.options.containerConfig.User,
		Privileged:   container.options.hostConfig.Privileged,
		Env:          container.options.containerConfig.Env,
	}
	execID, err := client.ContainerExecCreate(
		e.context,
		container.ID,
		execOpts,
	)
	if err != nil {
		return errors.Wrapf(err,
			"cannot create execution %v in container", strings.Join(cmd, " "))
	}
	err = client.ContainerExecStart(
		e.context,
		execID.ID,
		types.ExecStartCheck{
			Detach: true,
			Tty:    isTty,
		},
	)
	if err != nil {
		return errors.Wrapf(err,
			"cannot start execution %v in container", strings.Join(cmd, " "))
	}

	attachOpts := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  e.Stdin != nil,
		Stdout: true,
		Stderr: true,
	}

	resp, errAttach := client.ContainerAttach(
		e.context,
		container.ID,
		attachOpts,
	)
	if errAttach != nil && err != httputil.ErrPersistEOF {
		// ContainerAttach returns an ErrPersistEOF (connection closed)
		// means server met an error and put it in Hijacked connection
		// keep the error and read detailed error message from hijacked connection later
		return errors.Wrap(err, "cannot attach to container")
	}
	defer resp.Close()

	strm := &stream{
		stdin:  e.Stdin,
		stdout: e.Stdout,
		stderr: e.Stderr,
	}
	cErr := promise.Go(func() error {
		errHijack := holdHijackedConnection(
			e.context,
			strm,
			isTty,
			e.Stdin,
			e.Stdout,
			e.Stderr,
			resp,
		)
		if errHijack == nil {
			return errAttach
		}
		return errHijack
	})

	e.wc = cErr
	e.isStarted = true

	return nil
}
func (e *Execution) StdinPipe() (io.WriteCloser, error) {
	if e.Stdin != nil {
		return nil, errors.New("Docker execution:: Stdin already set")
	}
	pr, pw := io.Pipe()
	e.Stdin = pr
	return pw, nil
}

func (e *Execution) StderrPipe() (io.ReadCloser, error) {
	if e.Stderr != nil {
		return nil, errors.New("Docker execution Stderr already set")
	}
	pr, pw := io.Pipe()
	e.Stderr = pw
	e.closeAfterWait = append(e.closeAfterWait, pw)
	return pr, nil
}

func (e *Execution) StdoutPipe() (io.ReadCloser, error) {
	if e.Stderr != nil {
		return nil, errors.New("Docker execution stdout already set")
	}
	pr, pw := io.Pipe()
	e.Stdout = pw
	e.closeAfterWait = append(e.closeAfterWait, pw)
	return pr, nil
}
func (e *Execution) Wait() error {
	if e.isStarted == false {
		return nil
	}

	defer closeFds(e.closeAfterWait)

	if err := <-e.wc; err != nil {
		return errors.Wrap(err, "failed to wait for hijacked connection")
	}

	client := e.container.client
	inspect := func() error {
		info, err := client.ContainerExecInspect(e.context, e.execID)
		if err != nil {
			return err
		}
		if !info.Running {
			return nil
		}
		return errors.New("container is running")
	}
	return backoff.Retry(inspect, backoff.NewExponentialBackOff())
}

func closeFds(l []io.Closer) {
	for _, fd := range l {
		fd.Close()
	}
}

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

			log.Debug("[hijack] End of stdout")
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
			log.Debug("[hijack] End of stdin")
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
			return err
		}
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			select {
			case err := <-receiveStdout:
				if err != nil {
					log.Debugf("Error receiveStdout: %s", err)
					return err
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
	return command.NewInStream(s.stdin)
}

func (s *stream) Out() *command.OutStream {
	return command.NewOutStream(s.stdout)
}

func (s *stream) Err() io.Writer {
	return s.stderr
}

// ExitError reports an unsuccessful exit by a command.
type ExitError struct {
	// ExitCode holds the non-zero exit code of the container
	ExitCode int

	// Stderr holds the standard error output from the command
	// if it *Cmd executed through Output() and Cmd.Stderr was not
	// set.
	Stderr []byte
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("Docker execution: exit status: %d", e.ExitCode)
}
