package inkscape

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/galihrivanto/runner"
)

const (
	defaultCmdName  = "inkscape"
	shellModeBanner = "Inkscape interactive shell mode"
)

// defines common errors in library
var (
	ErrCommandNotReady = errors.New("inkscape not available")
)

var (
	bufferPool = NewSizedBufferPool(5, 1024*1024)
	verbose    bool
)

// SetVerbose override verbosity
func SetVerbose(v bool) {
	verbose = v
}

type chanWriter struct {
	out chan []byte
}

func (w *chanWriter) Write(data []byte) (int, error) {
	w.out <- data

	return len(data), nil
}

// Proxy runs inkspace instance in background and
// provides mechanism to interfacing with running
// instance via stdin
type Proxy struct {
	options Options

	// context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	cmd *exec.Cmd

	// input and output
	stdin  io.WriteCloser
	stdout chan []byte
	stderr chan []byte
}

// runBackground run inkscape instance in background
func (p *Proxy) runBackground(ctx context.Context, commandPath string, vars ...string) error {
	args := []string{
		"--shell",
	}

	if len(vars) > 0 {
		args = append(args, vars...)
	}

	cmd := exec.CommandContext(ctx, commandPath, args...)
	cmd.Stdout = &chanWriter{p.stdout}
	cmd.Stderr = &chanWriter{p.stderr}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	p.stdin = stdin

	log.Println("proxy: run in background")
	return cmd.Run()
}

// Run start inkscape proxy
func (p *Proxy) Run(args ...string) error {
	commandPath, err := exec.LookPath(p.options.commandName)
	if err != nil {
		return err
	}

	log.Println("proxy:", commandPath)

	p.ctx, p.cancel = context.WithCancel(context.Background())

	go func() {
		runner.RunWithRetry(
			p.ctx,
			func(ctx context.Context) error {
				return p.runBackground(ctx, commandPath, args...)
			},
			runner.NewExponentialBackoffRetry(p.options.maxRetry),
		)
	}()

	return nil
}

// Close satisfy io.Closer interface
func (p *Proxy) Close() error {
	p.cancel()
	p.stdin.Close()
	close(p.stdout)
	close(p.stderr)

	return nil
}

// waitReady wait until background process
// ready accepting command
func (p *Proxy) waitReady(timeout time.Duration) error {
	ready := make(chan struct{})
	go func() {
		for {
			// query stdin availability every second
			if p.stdin != nil {
				close(ready)
				return
			}

			<-time.After(time.Second)
		}
	}()

	select {
	case <-time.After(timeout):
		return ErrCommandNotReady
	case <-ready:
		return nil
	}
}

func (p *Proxy) sendCommand(b []byte) ([]byte, error) {
	log.Println("proxy: wait ready")
	err := p.waitReady(5 * time.Second)
	if err != nil {
		return nil, err
	}

	log.Println("proxy: send command to stdin", string(b))

	// append new line
	if !bytes.HasSuffix(b, []byte{'\n'}) {
		b = append(b, '\n')
	}

	_, err = p.stdin.Write(b)
	if err != nil {
		return nil, err
	}

	// wait output
	var output []byte

waitLoop:
	for {
		select {
		case bytesErr := <-p.stderr:
			// for now, we can only check error message pattern
			// ignore WARNING
			if bytes.Contains(output, []byte("WARNING")) {
				log.Println(string(bytesErr))
				break
			}

			err = fmt.Errorf("%s", string(bytesErr))
			break waitLoop
		case output = <-p.stdout:
			log.Println(string(output))
			if len(output) == 0 {
				break
			}

			// check if shell mode banner
			if bytes.Contains(output, []byte(shellModeBanner)) {
				break
			}

			break waitLoop
		}
	}

	return output, nil
}

// RawCommands send inkscape shell commands
func (p *Proxy) RawCommands(args ...string) ([]byte, error) {
	buffer := bufferPool.Get()
	defer bufferPool.Put(buffer)

	// construct command buffer
	buffer.WriteString(strings.Join(args, ";"))

	return p.sendCommand(buffer.Bytes())
}

// Svg2Pdf convert svg input file to output pdf file
func (p *Proxy) Svg2Pdf(svgIn, pdfOut string) error {
	res, err := p.RawCommands(
		"file-open:"+svgIn,
		"export-filename:"+pdfOut,
		"export-do",
	)
	if err != nil {
		return err
	}

	log.Println("proxy: result", string(res))

	return nil
}

// NewProxy create new inkscape proxy instance
func NewProxy(opts ...Option) *Proxy {
	// default value
	init := Options{
		commandName: defaultCmdName,
		maxRetry:    5,
		verbose:     false,
	}

	// merge options
	options := mergeOptions(init, opts...)

	// check verbosity
	if !options.verbose {
		log.SetOutput(ioutil.Discard)
	}

	stdout := make(chan []byte)
	stderr := make(chan []byte)

	return &Proxy{
		options: options,
		stdout:  stdout,
		stderr:  stderr,
	}
}
