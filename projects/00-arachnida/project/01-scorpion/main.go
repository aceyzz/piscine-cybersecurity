package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	usage = "usage: ./scorpion FILE1 [FILE2 ...]"
	help  = usage + `

Display file attributes and metadata of images (.jpg .jpeg .png .gif .bmp).
Parsed metadata: EXIF, JFIF, XMP presence, JPEG comments, PNG chunks, GIF and BMP headers.

options:
  -h       show this help
`
)

func main() {
	files, err := parseArgs(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		fmt.Print(help)
		return
	}
	if err != nil {
		warn(err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if !showAll(files) {
		os.Exit(1)
	}
}

func parseArgs(args []string) ([]string, error) {
	fs := flag.NewFlagSet("scorpion", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() == 0 {
		return nil, errors.New("at least one file expected")
	}
	return fs.Args(), nil
}

func showAll(files []string) bool {
	ok := true
	for i, path := range files {
		if i > 0 {
			fmt.Println()
		}
		r, err := buildReport(path)
		if err != nil {
			warn(err)
			ok = false
			continue
		}
		printReport(r)
	}
	return ok
}
