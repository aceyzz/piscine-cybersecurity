package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
)

const (
	defaultDepth  = 5
	defaultOutput = "./data/"
	usage         = "usage: ./spider [-r] [-l N] [-p PATH] URL"
	help          = usage + `

Download images (.jpg .jpeg .png .gif .bmp) from a website.

options:
  -r       download images recursively from linked pages
  -l N     maximum recursion depth, requires -r (default 5)
  -p PATH  output directory (default ./data/)
  -h       show this help
`
)

type config struct {
	start  *url.URL
	depth  int
	output string
}

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		fmt.Print(help)
		return
	}
	if err != nil {
		warn(err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err := os.MkdirAll(cfg.output, 0o755); err != nil {
		warn(err)
		os.Exit(1)
	}
	saved := newCrawler(cfg).run()
	fmt.Printf("%d image(s) saved in %s\n", saved, cfg.output)
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("spider", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	recursive := fs.Bool("r", false, "")
	depth := fs.Int("l", defaultDepth, "")
	output := fs.String("p", defaultOutput, "")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 1 {
		return config{}, errors.New("exactly one URL expected")
	}
	if isFlagSet(fs, "l") && !*recursive {
		return config{}, errors.New("-l requires -r")
	}
	if *depth < 0 {
		return config{}, errors.New("depth must be >= 0")
	}
	start, err := parseHTTPURL(fs.Arg(0))
	if err != nil {
		return config{}, err
	}
	cfg := config{start: start, output: *output}
	if *recursive {
		cfg.depth = *depth
	}
	return cfg, nil
}
