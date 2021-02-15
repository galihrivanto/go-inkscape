package inkscape

import (
	"fmt"
	"testing"
)

func TestConcurrent(t *testing.T) {
	tempFiles := make([]string, 0)
	// defer func() {
	// 	for _, t := range tempFiles {
	// 		os.Remove(t)
	// 	}
	// }()

	proxy := NewProxy(Verbose(true))
	proxy.Run()

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
