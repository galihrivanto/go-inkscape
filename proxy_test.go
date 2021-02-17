package inkscape

import (
	"fmt"
	"os"
	"testing"
)

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
