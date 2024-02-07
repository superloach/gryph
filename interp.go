package gryph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

// Interp is a handle to a Python interpreter.
type Interp struct {
	cmd *exec.Cmd

	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// Option is a modifier which can be passed to NewInterp.
type Option func(*Interp) error

// Command is a command sent over the interpreter's stdin.
// Various commands are supported, such as getting and setting values,
// running scripts, and more.
type Command struct {
	Type string   `json:"type"`
	Var string   `json:"var"`
	Value interface{} `json:"value"`
}

func GetterCmd(varname string) Command {
	return Command{
		Type: "get",
		Var: varname,
	}
}

func SetterCmd(varname string, value interface{}) (Command, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(value); err != nil {
		return Command{}, fmt.Errorf("encode value: %w", err)
	}

	return Command{
		Type: "set",
		Args: []string{varname, buf.String()},
	}, nil
}

var defaultOptions = []Option{
	// interpreter startup script
	WithArgs("-c", `
	import sys, json

	# interpreter value table
	values = dict()
	values['_'] = values

	# handle commands from stdin
	def handle_command(cmd):
		if cmd['type'] == 'get':
			return {
				'type': 'get',
				'var': cmd['var'],
				'value': values[cmd['var']]
			}
		elif cmd['type'] == 'set':
			value = json.loads(cmd['value'])
			values[cmd['var']] = value
			return None
		else:
			raise ValueError('invalid command type')

	while True:
		# read command from stdin
		cmd = json.load(sys.stdin)

		# handle command
		resp = handle_command(cmd)

		# write response to stdout
		json.dump(resp, sys.stdout)
		sys.stdout.flush()
	`),

	// set up the interpreter's pipes
	func(interp *Interp) error {
		stdin, err := interp.cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("get stdin pipe: %w", err)
		}

		stdout, err := interp.cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("get stdout pipe: %w", err)
		}

		stderr, err := interp.cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("get stderr pipe: %w", err)
		}

		interp.stdin = stdin
		interp.stdout = stdout
		interp.stderr = stderr

		return nil
	},
}

// WithArgs sets the command-line arguments for the interpreter.
func WithArgs(args ...string) Option {
	return func(interp *Interp) error {
		interp.cmd.Args = append(interp.cmd.Args, args...)
		return nil
	}
}

// WithPath sets the path to the interpreter.
func WithPath(path string) Option {
	return func(interp *Interp) error {
		interp.cmd.Path = path
		return nil
	}
}

// WithEnv sets the environment for the interpreter.
func WithEnv(env []string) Option {
	return func(interp *Interp) error {
		interp.cmd.Env = env
		return nil
	}
}

// NewInterp creates a new Python interpreter.
func NewInterp(opts ...Option) (*Interp, error) {
	interp := &Interp{
		cmd: exec.Command("python"),
	}

	for i, opt := range append(defaultOptions, opts...) {
		if err := opt(interp); err != nil {
			return nil, fmt.Errorf("option %d failed: %w", i+1, err)
		}
	}

	return interp, nil
}

// Start starts the interpreter.
func (interp *Interp) Start() error {
	return interp.cmd.Start()
}

// Wait waits for the interpreter to exit.
func (interp *Interp) Wait() error {
	return interp.cmd.Wait()
}

// Close closes the interpreter.
func (interp *Interp) Close() error {
	return interp.cmd.Process.Kill()
}

// Run runs a Python script in the interpreter.
func (interp *Interp) Run(script string) (string, error) {
	// pass the script to the interpreter's stdin
	if _, err := interp.stdin.Write([]byte(script + "\n")); err != nil {
		return "", fmt.Errorf("write to stdin: %w", err)
	}

	// read the interpreter's stdout
	out := make([]byte, 1024)
	n, err := interp.stdout.Read(out)
	if err != nil {
		return "", fmt.Errorf("read from stdout: %w", err)
	}

	return string(out[:n]), nil
}
