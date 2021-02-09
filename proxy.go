package inkscape

import (
	"bufio"
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

	// // pipe stderr
	// stderrC := make(chan []byte)
	// defer close(stderrC)

	// cmd.Stderr = &chanWriter{out: stderrC}

	// pipe stdout
	stdoutC := make(chan []byte)
	defer close(stdoutC)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdoutReader := bufio.NewReader(stdout)
	go func() {
		for {

		}
	}()

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

		// case byteErr := <-stderrC:
		// 	if len(byteErr) == 0 {
		// 		break
		// 	}

		// 	if bytes.Contains(byteErr, []byte("WARNING")) {
		// 		continue
		// 	}

		// 	p.stderr <- byteErr

		case byteOut := <-stdoutC:
			if len(byteOut) == 0 {
				debug("new line, why?")
				break
			}

			// check if shell mode banner
			if bytes.Contains(byteOut, []byte(shellModeBanner)) {
				debug(string(byteOut))
				break
			}

			p.stdout <- byteOut
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

	return nil
}

// Close satisfy io.Closer interface
func (p *Proxy) Close() error {
	// send quit command
	_, err := p.RawCommands(quitCommand)

	p.cancel()
	close(p.requestLimiter)
	close(p.requestQueue)
	close(p.stderr)
	close(p.stdout)

	return err
}

func (p *Proxy) sendCommand(b []byte) ([]byte, error) {
	debug("send command to stdin", string(b))

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

waitLoop:
	for {
		select {
		case bytesErr := <-p.stderr:

			err = fmt.Errorf("%s", string(bytesErr))
			break waitLoop
		case bytesOut := <-p.stdout:
			fmt.Println(string(bytesOut))

			// TODO: use sentinel
			output = bytesOut

			break waitLoop
		}
	}

	return output, err
}

// RawCommands send inkscape shell commands
func (p *Proxy) RawCommands(args ...string) ([]byte, error) {
	buffer := bufferPool.Get()
	defer bufferPool.Put(buffer)

	// wait available
	debug("wait available")
	<-p.requestLimiter

	// construct command buffer
	buffer.WriteString(strings.Join(args, ";"))

	res, err := p.sendCommand(buffer.Bytes())

	// make it available again
	p.requestLimiter <- struct{}{}

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

	<-time.After(30 * time.Second)

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

func drain(c chan []byte) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}
