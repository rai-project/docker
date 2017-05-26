package docker

import (
	"errors"
	"io"
	"os"
	"runtime"

	"github.com/docker/docker/pkg/term"
)

// OutStream is an output stream used by the DockerCli to write normal program
// output.
type OutStream struct {
	out        io.Writer
	fd         uintptr
	isTerminal bool
	state      *term.State
}

func (o *OutStream) Write(p []byte) (int, error) {
	return o.out.Write(p)
}

// FD returns the file descriptor number for this stream
func (o *OutStream) FD() uintptr {
	return o.fd
}

// IsTerminal returns true if this stream is connected to a terminal
func (o *OutStream) IsTerminal() bool {
	return o.isTerminal
}

// SetRawTerminal sets raw mode on the output terminal
func (o *OutStream) SetRawTerminal() (err error) {
	if os.Getenv("NORAW") != "" || !o.isTerminal {
		return nil
	}
	o.state, err = term.SetRawTerminalOutput(o.fd)
	return err
}

// RestoreTerminal restores normal mode to the terminal
func (o *OutStream) RestoreTerminal() {
	if o.state != nil {
		term.RestoreTerminal(o.fd, o.state)
	}
}

// GetTtySize returns the height and width in characters of the tty
func (o *OutStream) GetTtySize() (uint, uint) {
	if !o.isTerminal {
		return 0, 0
	}
	ws, err := term.GetWinsize(o.fd)
	if err != nil {
		log.Debugf("Error getting size: %s", err)
		if ws == nil {
			return 0, 0
		}
	}
	return uint(ws.Height), uint(ws.Width)
}

// NewOutStream returns a new OutStream object from a Writer
func NewOutStream(out io.Writer) *OutStream {
	fd, isTerminal := term.GetFdInfo(out)
	return &OutStream{out: out, fd: fd, isTerminal: isTerminal}
}

// InStream is an input stream used by the DockerCli to read user input
type InStream struct {
	in         io.ReadCloser
	fd         uintptr
	isTerminal bool
	state      *term.State
}

func (i *InStream) Read(p []byte) (int, error) {
	return i.in.Read(p)
}

// Close implements the Closer interface
func (i *InStream) Close() error {
	return i.in.Close()
}

// FD returns the file descriptor number for this stream
func (i *InStream) FD() uintptr {
	return i.fd
}

// IsTerminal returns true if this stream is connected to a terminal
func (i *InStream) IsTerminal() bool {
	return i.isTerminal
}

// SetRawTerminal sets raw mode on the input terminal
func (i *InStream) SetRawTerminal() (err error) {
	if os.Getenv("NORAW") != "" || !i.isTerminal {
		return nil
	}
	i.state, err = term.SetRawTerminal(i.fd)
	return err
}

// RestoreTerminal restores normal mode to the terminal
func (i *InStream) RestoreTerminal() {
	if i.state != nil {
		term.RestoreTerminal(i.fd, i.state)
	}
}

// CheckTty checks if we are trying to attach to a container tty
// from a non-tty client input stream, and if so, returns an error.
func (i *InStream) CheckTty(attachStdin, ttyMode bool) error {
	// In order to attach to a container tty, input stream for the client must
	// be a tty itself: redirecting or piping the client standard input is
	// incompatible with `docker run -t`, `docker exec -t` or `docker attach`.
	if ttyMode && attachStdin && !i.isTerminal {
		eText := "the input device is not a TTY"
		if runtime.GOOS == "windows" {
			return errors.New(eText + ".  If you are using mintty, try prefixing the command with 'winpty'")
		}
		return errors.New(eText)
	}
	return nil
}

// NewInStream returns a new InStream object from a ReadCloser
func NewInStream(in io.ReadCloser) *InStream {
	fd, isTerminal := term.GetFdInfo(in)
	return &InStream{in: in, fd: fd, isTerminal: isTerminal}
}

// Streams is an interface which exposes the standard input and output streams
type Streams interface {
	In() *InStream
	Out() *OutStream
	Err() io.Writer
}

type stream struct {
	stdin  io.ReadCloser
	stdout io.Writer
	stderr io.Writer
}

func (s *stream) In() *InStream {
	if s.stdin == nil {
		return nil
	}
	return NewInStream(s.stdin)
}

func (s *stream) Out() *OutStream {
	if s.stdout == nil {
		return nil
	}
	return NewOutStream(s.stdout)
}

func (s *stream) Err() io.Writer {
	return s.stderr
}
