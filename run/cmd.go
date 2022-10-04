package run

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
)

// Cmd is a wrapper for *exec.Cmd that adds output-capturing to its Run method.
type Cmd struct {
	*exec.Cmd

	// Populated by Shell if parsing fails.
	err error
}

// NewCmd creates a new Cmd exactly like exec.CommandContext.
func NewCmd(ctx context.Context, cmdname string, args ...string) Cmd {
	return Cmd{Cmd: exec.CommandContext(ctx, cmdname, args...)}
}

// Shell creates a new Cmd by parsing str -
// a command and its arguments together -
// as if by a command shell.
//
// If str cannot be parsed,
// an error is stored in Cmd and returned when Run is called.
// This is to make it convenient to call Shell(...).Run().
func Shell(ctx context.Context, str string) Cmd {
	words, err := shellwords.Parse(str)
	if err != nil {
		return Cmd{err: err}
	}
	if len(words) == 0 {
		return Cmd{err: err}
	}
	return NewCmd(ctx, words[0], words[1:]...)
}

// Shellf is like Shell but the string is a Sprintf format filled in with the remaining args.
func Shellf(ctx context.Context, str string, args ...any) Cmd {
	return Shell(ctx, fmt.Sprintf(str, args...))
}

// Run runs the command.
// It captures any stdout/stderr output that would otherwise be discarded,
// and wraps it up in an Error in case of error.
func (c Cmd) Run() error {
	if c.err != nil {
		return c.err
	}

	var buf *bytes.Buffer
	if c.Stdout == nil || c.Stderr == nil {
		buf = new(bytes.Buffer)
		if c.Stdout == nil {
			c.Stdout = buf
		}
		if c.Stderr == nil {
			c.Stderr = buf
		}
	}
	err := c.Cmd.Run()
	if err != nil && buf != nil {
		return Error{
			Cmd:    c,
			Err:    err,
			Output: buf.String(),
		}
	}
	return err
}

func (c Cmd) Start() error {
	if c.err != nil {
		return c.err
	}
	return c.Cmd.Start()
}

func (c Cmd) Output() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.Cmd.Output()
}

func (c Cmd) CombinedOutput() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.Cmd.CombinedOutput()
}

// Error is the error type returned by Cmd.Run
// when it has captured stdout/stderr contents
// that would otherwise be discarded.
type Error struct {
	Cmd    Cmd
	Err    error
	Output string
}

func (e Error) Error() string {
	e.Output = strings.TrimRight(e.Output, "\r\n")
	if len(e.Output) == 0 {
		return e.Err.Error()
	}
	return fmt.Sprintf("error from `%s` (output follows): %s\n%s", strings.Join(e.Cmd.Args, " "), e.Err, e.Output)
}

func (e Error) Unwrap() error {
	return e.Err
}
