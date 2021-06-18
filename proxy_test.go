package inkscape

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

// TestConcurrent tests the library usage in concurrent environment
func TestConcurrent(t *testing.T) {
	tempFiles := make([]string, 0)
	defer func() {
		for _, t := range tempFiles {
			os.Remove(t)
		}
	}()

	proxy := NewProxy(Verbose(true))
	err := proxy.Run()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer proxy.Close()

	for i := 0; i < 10; i++ {
		temp := fmt.Sprintf("%d.pdf", i)
		tempFiles = append(tempFiles, temp)

		go func() {
			if err := proxy.Svg2Pdf("circle.svg", temp); err != nil {
				t.Error(err)
			}
		}()
	}
}

// TestExecContext tests against command execution within context boundary
func TestExecContext(t *testing.T) {
	const file = "circle.svg"
	n := rand.Intn(1000)
	tmpFile := fmt.Sprintf("%s.tmp.%d.pdf", file, n)
	defer func() {
		os.Remove(tmpFile)
	}()

	proxy := NewProxy(Verbose(true))
	err := proxy.Run()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer proxy.Close()

	// gives very short life of execution context to tests
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1)
	done := make(chan struct{})
	defer cancel()

	go func() {
		// this command expected to run no more than specified timeout duration
		err := proxy.Svg2PdfContext(ctx, file, tmpFile)
		if err != nil {
			if err != ErrCommandExecCanceled {
				t.Error(err)
			}
		}
		if err == nil {
			t.Error("expected command to be canceled, got success command execution")
		}
		done <- struct{}{}
	}()

	<-done
}

func TestOpenFail(t *testing.T) {
	proxy := NewProxy(Verbose(true))
	err := proxy.Run()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer proxy.Close()

	res, err := proxy.RawCommands("file-open:notexists.svg")
	if err == nil {
		t.Error("should fail", string(res))
		t.FailNow()
	}

	t.Log(string(res))
}
