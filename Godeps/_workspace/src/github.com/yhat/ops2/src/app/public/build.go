package main

// have a `go generate` rule so boxr triggers a make
//go:generate make

import (
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatalf("usage: build.go 'template file' 'content'")
	}

	data, err := ioutil.ReadFile(args[0])
	if err != nil {
		log.Fatal(err)
	}
	content, err := ioutil.ReadFile(args[1])
	if err != nil {
		log.Fatal(err)
	}

	tmpl, err := template.New("base").Parse(string(data))
	if err != nil {
		log.Fatalf("failed to parse template file '%s' %v", args[0], err)
	}

	tmpl.Execute(os.Stdout, map[string]string{"Content": string(content)})
}
