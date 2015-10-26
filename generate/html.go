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
func SearchHTML(r io.Reader, abs bool) ([]*http2preload.Asset, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return searchNodes(doc, abs), nil
}

func searchNodes(root *html.Node, abs bool) []*http2preload.Asset {
	var assets []*http2preload.Asset
	for n := root; n != nil; n = n.NextSibling {
		var a *http2preload.Asset
		switch n.DataAtom {
		case atom.Link:
			rel := getAttr("rel", n.Attr)
			if rel != relStylesheet && rel != relImport {
				break
			}
			a = &http2preload.Asset{URL: getAttr("href", n.Attr)}
			if rel == relStylesheet {
				a.Type = http2preload.Style
			}
		case atom.Script:
			a = &http2preload.Asset{URL: getAttr("src", n.Attr), Type: http2preload.Script}
		case atom.Img:
			a = &http2preload.Asset{URL: getAttr("src", n.Attr), Type: http2preload.Image}
		}
		if a != nil {
			if a.URL != "" && (abs || !isAbs(a.URL)) {
				assets = append(assets, a)
			}
			continue
		}
		assets = append(assets, searchNodes(n.FirstChild, abs)...)
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
