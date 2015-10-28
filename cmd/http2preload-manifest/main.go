// Copyright 2015 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This program generates preload manifest in JSON format.
// Install it with
//
//     go get -u github.com/google/http2preload/cmd/http2preload-manifest
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/http2preload"
	"github.com/google/http2preload/generate"
)

var (
	idxfile = flag.String("i", "index.html", "index file name to replace with /")
	noext   = flag.Bool("noext", true, "remove file extension from manifest keys")
	outname = flag.String("o", "", "manifest contents output; empty argument writes to stdout")
	strip   = flag.String("strip", "", "strip prefix from manifest keys")
)

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	*idxfile = strings.TrimPrefix(*idxfile, "/")

	out := os.Stdout
	if *outname != "" {
		var err error
		out, err = os.Create(*outname)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
	}

	// collect all source paths
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}
	var src []string
	for _, a := range args {
		src = append(src, walk(a)...)
	}
	src = unique(src)

	// create manifest from sources
	manifest := http2preload.Manifest{}
	type result struct {
		src string
		a   []*http2preload.Asset
		err error
	}
	n := len(src)
	if n > 100 {
		n = 100
	}
	results := make(chan result, n)
	for _, s := range src {
		go func() {
			r, err := os.Open(s)
			if err != nil {
				results <- result{src: s, err: err}
				return
			}
			a, err := generate.SearchHTML(r, false)
			results <- result{s, a, err}
		}()
	}
	for _ = range src {
		r := <-results
		if r.err != nil {
			log.Printf("%s: %v", r.src, r.err)
			continue
		}
		manifest[normPath(r.src)] = r.a
	}

	// output result
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := out.Write(b); err != nil {
		log.Fatal(err)
	}
}

func walk(root string) []string {
	if fi, err := os.Stat(root); err == nil && !fi.IsDir() {
		return []string{root}
	}
	var files []string
	filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%s: %v", p, err)
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		ext := filepath.Ext(p)
		if ext == ".html" || ext == ".htm" {
			files = append(files, p)
		}
		return nil
	})
	return files
}

func normPath(p string) string {
	p = strings.TrimPrefix(p, *strip)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if strings.HasSuffix(p, "/"+*idxfile) {
		p = strings.TrimSuffix(p, *idxfile)
	}
	if *noext {
		p = strings.TrimSuffix(p, filepath.Ext(p))
	}
	return p
}

func unique(items []string) []string {
	seen := make(map[string]bool, len(items))
	res := make([]string, 0, len(items))
	for _, s := range items {
		if !seen[s] {
			res = append(res, s)
			seen[s] = true
		}
	}
	return res
}

func usage() {
	fmt.Fprint(os.Stderr, usageText)
	flag.PrintDefaults()
}

const usageText = `Usage: http2preload-manifest [options] [src [src ...]]

The program scans for files with .html extension in locations specified by src
and generates a preload manifest file in JSON format, either writing it
to stdout or a file specified with -o argument.

If no src argument is provided, current directory will be used.

Options:
`
