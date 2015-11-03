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

// Package generate parses HTML and extracts assets suitable for
// constructing http2preload.Manifest.
package generate

import (
	"io"
	"strings"

	"github.com/google/http2preload"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	relStylesheet = "stylesheet"
	relImport     = "import"
)

// SearchHTML finds links to assets referenced in HTML read from r.
// Currently, only <img>, <link> and <script> tags are supported.
// abs argument will discard assets with absolute URLs.
func SearchHTML(r io.Reader, abs bool) (map[string]http2preload.AssetOpt, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return searchNodes(doc, abs), nil
}

func searchNodes(root *html.Node, abs bool) map[string]http2preload.AssetOpt {
	assets := make(map[string]http2preload.AssetOpt)
	for n := root; n != nil; n = n.NextSibling {
		var (
			url string
			opt http2preload.AssetOpt
		)
		switch n.DataAtom {
		case atom.Link:
			rel := getAttr("rel", n.Attr)
			if rel != relStylesheet && rel != relImport {
				break
			}
			url = getAttr("href", n.Attr)
			if rel == relStylesheet {
				opt.Type = http2preload.Style
			}
		case atom.Script:
			url = getAttr("src", n.Attr)
			opt.Type = http2preload.Script
		case atom.Img:
			url = getAttr("src", n.Attr)
			opt.Type = http2preload.Image
		}
		if url != "" {
			if abs || !isAbs(url) {
				assets[url] = opt
			}
			continue
		}
		for k, v := range searchNodes(n.FirstChild, abs) {
			assets[k] = v
		}
	}
	return assets
}

func getAttr(name string, attr []html.Attribute) string {
	for _, a := range attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func isAbs(u string) bool {
	return strings.HasPrefix(u, "http:") || strings.HasPrefix(u, "https:")
}
