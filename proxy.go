package inkscape

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/galihrivanto/runner"
)

const (
	defaultCmdName  = "inkscape"
	shellModeBanner = "Inkscape interactive shell mode"
	quitCommand     = "quit"
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

	log.Print(append([]interface{}{"proxy:"}, v...)...)
}

type chanWriter struct {
	out chan []byte
}

func (w *chanWriter) Write(data []byte) (int, error) {

	// look like the buffer being reused internally by the exec.Command
	// so we can directly read the buffer in another goroutine while still being used in exec.Command goroutine

	// copy to be written buffer and pass it into channel
	bufferToWrite := make([]byte, len(data))
	written := copy(bufferToWrite, data)
	w.out <- bufferToWrite

	return written, nil
}

// Proxy runs inkspace instance in background and
// provides mechanism to interfacing with running
// instance via stdin
type Proxy struct {
	options Options

	// context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// limiter to allow one command processed at time
	requestLimiter chan struct{}

	// queue of request
	requestQueue chan []byte

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

	// pipe stderr
	stderrC := make(chan []byte)
	defer close(stderrC)

	cmd.Stderr = &chanWriter{out: stderrC}

	// pipe stdout
	stdoutC := make(chan []byte)
	defer close(stdoutC)

	cmd.Stdout = &chanWriter{out: stdoutC}

	// pipe stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	// start command and wait it close
	debug("run in background")
	if err := cmd.Start(); err != nil {
		return err
	}

	// make first command available
	// after received prompt
wait:
	for {
		bytesOut := <-stdoutC
		bytesOut = bytes.TrimSpace(bytesOut)
		parts := bytes.Split(bytesOut, []byte("\n"))
		for _, part := range parts {
			if isPrompt(part) {
				break wait
			}
		}
	}

	select {
	case p.requestLimiter <- struct{}{}:
	default:
		// discard
	}

	// handle command and output
	for {
		select {
		case <-ctx.Done():
			return cmd.Wait()

		case command := <-p.requestQueue:
			debug("write command ", string(command))
			if _, err := stdin.Write(command); err != nil {
				p.stderr <- []byte(err.Error())
			}

		case bytesErr := <-stderrC:
			if len(bytesErr) == 0 {
				break
			}

			if bytes.Contains(bytesErr, []byte("WARNING")) {
				break
			}

			p.stderr <- bytes.TrimSpace(bytesErr)

		case bytesOut := <-stdoutC:
			if len(bytesOut) == 0 {
				break
			}

			p.stdout <- bytes.TrimSpace(bytesOut)
		}
	}
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

	// print inkscape version
	res, _ := p.RawCommands(Version())
	fmt.Println(string(res))

	return nil
}

// Close satisfy io.Closer interface
func (p *Proxy) Close() error {
	// send quit command
	_, err := p.sendCommand([]byte(quitCommand), false)

	p.cancel()
	close(p.requestLimiter)
	close(p.requestQueue)
	close(p.stderr)
	close(p.stdout)

	return err
}

func (p *Proxy) sendCommand(b []byte, waitPrompt ...bool) ([]byte, error) {
	wait := true
	if len(waitPrompt) > 0 {
		wait = waitPrompt[0]
	}

	// wait available
	debug("wait prompt available")
	<-p.requestLimiter
	defer func() {
		// make it available again
		p.requestLimiter <- struct{}{}
	}()

	debug("send command to stdin ", string(b))

	// drain old err and out
	drain(p.stderr)
	drain(p.stdout)

	// append new line
	if !bytes.HasSuffix(b, []byte{'\n'}) {
		b = append(b, '\n')
	}

	p.requestQueue <- b

	var (
		output []byte
		err    error
	)

	// immediate return
	if !wait {
		<-time.After(time.Second)
		return []byte{}, nil
	}

waitLoop:
	for {
		// wait until received prompt
		bytesOut := <-p.stdout
		debug(string(bytesOut))
		parts := bytes.Split(bytesOut, []byte("\n"))
		for _, part := range parts {
			if isPrompt(part) {
				break waitLoop
			}
		}

		output = append(output, bytesOut...)
	}

	// drain error channel
errLoop:
	for {
		select {
		case bytesErr := <-p.stderr:
			if len(bytesErr) > 0 {
				debug(string(bytesErr))
				err = fmt.Errorf("%s", string(bytesErr))
			}
		default:
			break errLoop
		}
	}

	return output, err
}

// RawCommands send inkscape shell commands
func (p *Proxy) RawCommands(args ...string) ([]byte, error) {
	buffer := bufferPool.Get()
	defer bufferPool.Put(buffer)

	// construct command buffer
	buffer.WriteString(strings.Join(args, ";"))

	res, err := p.sendCommand(buffer.Bytes())

	return res, err
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
	options := Options{
		commandName: defaultCmdName,
		maxRetry:    5,
		verbose:     false,
	}

	// merge options
	options = mergeOptions(options, opts...)

	// check verbosity
	verbose = options.verbose

	return &Proxy{
		options: options,
		stdout:  make(chan []byte, 100),
		stderr:  make(chan []byte, 100),

		// limit request to one request at time
		requestLimiter: make(chan struct{}, 1),
		requestQueue:   make(chan []byte, 100),
	}
}

func isPrompt(data []byte) bool {
	return bytes.Equal(data, []byte(">"))
}

func drain(c chan []byte) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}
