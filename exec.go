package docker

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	dc "github.com/fsouza/go-dockerclient"
	shellwords "github.com/junegunn/go-shellwords"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type Execution struct {
	container *Container
	cancel    context.CancelFunc

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
	Stdin io.Reader

	// Stdout and Stderr specify the process's standard output and error.
	// If either is nil, they will be redirected to the null device (os.DevNull).
	//
	// Run will not close the underlying handles if they are *os.File differently
	// than os/exec.
	Stdout io.Writer
	Stderr io.Writer

	started        bool
	exec           *dc.Exec
	cw             dc.CloseWaiter
	closeAfterWait []io.Closer
}

func NewExecution(container *Container, args ...string) (*Execution, error) {
	return &Execution{
		container: container,
		Path:      args[0],
		Args:      args[1:],
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

// CombinedOutput runs the command and returns its combined standard output and
// standard error.
//
// Docker API does not have strong guarantees over ordering of messages. For instance:
//     >&1 echo out; >&2 echo err
// may result in "out\nerr\n" as well as "err\nout\n" from this method.
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

// Output runs the command and returns its standard output.
//
// If the container exits with a non-zero exit code, the error is of type
// *ExitError. Other error types may be returned for I/O problems and such.
//
// If c.Stderr was nil, Output populates ExitError.Stderr.
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

// Run starts the specified command and waits for it to complete.
//
// If the command runs successfully and copying streams are done as expected,
// the error is nil.
//
// If the container exits with a non-zero exit code, the error is of type
// *ExitError. Other error types may be returned for I/O problems and such.
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
		e.Stdin = empty
	}
	if e.Stdout == nil {
		e.Stdout = ioutil.Discard
	}
	if e.Stderr == nil {
		e.Stderr = ioutil.Discard
	}

	cmd := append([]string{e.Path}, e.Args...)

	c1, err := client.InspectContainer(container.ID)
	if err != nil {
		err = Error(err)
		return errors.Wrap(err, "Docker execution:: cannot inspect container")
	}
	containerConfig := c1.Config
	opts := dc.CreateExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          cmd,
		Container:    container.ID,
		User:         containerConfig.User,
		Privileged:   true,
	}

	exec, err := client.CreateExec(opts)
	if err != nil {
		err = Error(err)
		return errors.Wrap(err, "Docker execution:: failed to create docker execution")
	}

	cw, err := client.StartExecNonBlocking(
		exec.ID,
		dc.StartExecOptions{
			InputStream:  e.Stdin,
			OutputStream: e.Stdout,
			ErrorStream:  e.Stderr,
			Detach:       false,
			RawTerminal:  true,
			Tty:          true,
		},
	)
	if err != nil {
		err = Error(err)
		return errors.Wrap(err, "Docker execution:: failed to start docker execution")
	}
	e.exec = exec
	e.cw = cw
	return nil
}

// StdinPipe returns a pipe that will be connected to the command's standard input
// when the command starts.
//
// Different than os/exec.StdinPipe, returned io.WriteCloser should be closed by user.
func (e *Execution) StdinPipe() (io.WriteCloser, error) {
	if e.Stdin != nil {
		return nil, errors.New("Docker execution:: Stdin already set")
	}
	pr, pw := io.Pipe()
	e.Stdin = pr
	return pw, nil
}

// StdoutPipe returns a pipe that will be connected to the command's standard output when
// the command starts.
//
// Wait will close the pipe after seeing the command exit or in error conditions.
func (e *Execution) StdoutPipe() (io.ReadCloser, error) {
	if e.Stdout != nil {
		return nil, errors.New("Docker execution Stdout already set")
	}
	pr, pw := io.Pipe()
	e.Stdout = pw
	e.closeAfterWait = append(e.closeAfterWait, pw)
	return pr, nil
}

// StderrPipe returns a pipe that will be connected to the command's standard error when
// the command starts.
//
// Wait will close the pipe after seeing the command exit or in error conditions.
func (e *Execution) StderrPipe() (io.ReadCloser, error) {
	if e.Stderr != nil {
		return nil, errors.New("Docker execution Stderr already set")
	}
	pr, pw := io.Pipe()
	e.Stderr = pw
	e.closeAfterWait = append(e.closeAfterWait, pw)
	return pr, nil
}

func (e *Execution) Wait() error {

	container := e.container
	client := container.client

	defer closeFds(e.closeAfterWait)
	if e.cw == nil {
		return errors.New("not container is attached")
	}
	err := e.cw.Wait()
	if err != nil {
		return errors.Wrap(err, "Docker execution wait failed.")
	}
	inspect, err := client.InspectExec(e.exec.ID)
	if err != nil {
		err = Error(err)
		return errors.Wrap(err, "Docker execution: cannot wait for container")
	}
	if inspect.Running {
		return errors.Errorf("Docker execution: expecting the execution to have stopped")
	}
	if inspect.ExitCode != 0 {
		return &ExitError{ExitCode: inspect.ExitCode}
	}
	return nil
}

func closeFds(l []io.Closer) {
	for _, fd := range l {
		fd.Close()
	}
}

func (e *Execution) Cancel() {
	e.cancel()
}

type emptyReader struct{}

func (r *emptyReader) Read(b []byte) (int, error) {
	return 0, io.EOF
}

var empty = &emptyReader{}

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
