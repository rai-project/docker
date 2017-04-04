package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http/httputil"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/promise"
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

	var cmd string
	var cmdArgs []string

	if len(args) > 0 {
		cmd = args[0]
	}
	if len(args) > 1 {
		cmdArgs = args[1:]
	}
	return &Execution{
		container:  container,
		Path:       cmd,
		Args:       cmdArgs,
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

	env := e.Env
	if len(env) == 0 {
		env = container.options.containerConfig.Env
	}

	cmd := append([]string{e.Path}, e.Args...)
	execOpts := types.ExecConfig{
		AttachStdin:  e.Stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       false,
		Tty:          isTty,
		Cmd:          cmd,
		User:         container.options.containerConfig.User,
		Privileged:   container.options.hostConfig.Privileged,
		Env:          env,
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

	e.execID = execID.ID

	resp, errAttach := client.ContainerExecAttach(
		e.context,
		e.execID,
		execOpts,
	)
	if errAttach != nil && errAttach != httputil.ErrPersistEOF {
		// ContainerAttach returns an ErrPersistEOF (connection closed)
		// means server met an error and put it in Hijacked connection
		// keep the error and read detailed error message from hijacked connection later
		return errors.Wrap(errAttach, "cannot attach to container")
	}

	strm := &stream{
		stdin:  e.Stdin,
		stdout: e.Stdout,
		stderr: e.Stderr,
	}
	cErr := promise.Go(func() error {
		defer resp.Close()
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

	return err
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
	if !e.isStarted {
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
