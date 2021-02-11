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
	"sync"
	"time"

	"github.com/galihrivanto/runner"
)

const (
	defaultCmdName  = "inkscape"
	shellModeBanner = "Inkscape interactive shell mode"
)

// defines common errors in library
var (
	ErrCommandNotAvailable = errors.New("inkscape not available")
	ErrCommandNotReady     = errors.New("inkscape not ready")
)

var (
	bufferPool = NewSizedBufferPool(5, 1024*1024)
	verbose    bool
)

func debug(v ...interface{}) {
	if !verbose {
		return
	}

	log.Print("proxy:")
	log.Println(v...)
}

type chanWriter struct {
	out chan []byte
}

func (w *chanWriter) Write(data []byte) (int, error) {

	// look like the buffer being reused internally by the exec.Command
	// so we can directly read the buffer in another goroutine while still being used in exec.Command goroutine

	// copy to be written buffer and pass it into channel
	bufferToWrite := append([]byte{}, data...)
	w.out <- bufferToWrite

	return len(bufferToWrite), nil
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

	// input
	lock  sync.RWMutex
	stdin io.WriteCloser

	// output
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

	p.lock.Lock()
	p.stdin = stdin
	p.lock.Unlock()

	defer func() {
		// only close channel when command closes
		close(p.stdout)
		close(p.stderr)
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	debug("run in background")

	return cmd.Wait()
}

// Run start inkscape proxy
func (p *Proxy) Run(args ...string) error {
	commandPath, err := exec.LookPath(p.options.commandName)
	if err != nil {
		return ErrCommandNotAvailable
	}

	debug(commandPath)

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

	return nil
}

// waitReady wait until background process
// ready accepting command
func (p *Proxy) waitReady(timeout time.Duration) error {
	ready := make(chan struct{})
	go func() {
		for {
			// query stdin availability every second
			p.lock.RLock()
			if p.stdin != nil {
				p.lock.RUnlock()
				close(ready)
				return
			}
			p.lock.RUnlock()

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
	debug("wait ready")
	err := p.waitReady(30 * time.Second)
	if err != nil {
		return nil, err
	}

	debug("send command to stdin", string(b))

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
				debug(string(bytesErr))
				break
			}

			err = fmt.Errorf("%s", string(bytesErr))
			break waitLoop
		case output = <-p.stdout:
			if len(output) == 0 {
				break
			}

			// check if shell mode banner
			if bytes.Contains(output, []byte(shellModeBanner)) {
				debug(string(output))
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
		FileOpen(svgIn),
		ExportFileName(pdfOut),
		ExportDo(),
		FileClose(),
	)
	if err != nil {
		return err
	}

	debug("result", string(res))

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
