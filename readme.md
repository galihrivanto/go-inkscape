<div align="center">
  <h1>go-inkscape</h1>
  
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/galihrivanto/go-inkscape)
  
a proxy to interfacing with inkscape `--shell` mode 
</div>

# **motivation**
while golang have some great pdf libraries but currently there's no one support complete svg conversion to pdf. this library attempt to provide interfacing with wellknown **inkscape** to manipulate svg, pdf and other supported formats using `--shell` mode and `action-command`

# **install**
```bash
go get github.com/galihrivanto/go-inkscape
```

# **simple usage**
```go
func main() {
	var (
	svgInput  string
	pdfOutput string
)

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	flag.StringVar(&svgInput, "input", "", "svg input")
	flag.StringVar(&pdfOutput, "output", "result.pdf", "pdf output")
	flag.Parse()

	if svgInput == "" {
		fmt.Println("svg input is missing")
		os.Exit(1)
	}

	proxy := inkscape.NewProxy(inkscape.Verbose(true))
	err := proxy.Run()
	handleErr(err)
	defer proxy.Close()

	err = proxy.Svg2Pdf(svgInput, pdfOutput)
	handleErr(err)

	fmt.Println("done!!")
}
}
```

# **advanced usage**
```go
...
proxy.RawCommands(
    "file-open:test.svg",
    "export-filename:out.pdf",
    "export-do",
)
...

```

# license
[MIT](https://choosealicense.com/licenses/mit/)
