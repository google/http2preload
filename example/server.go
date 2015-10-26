package app

import (
	"fmt"
	"net/http"

	"github.com/google/http2preload"
)

const gopher = "/gopher.png"

//go:generate http2preload-manifest -o preload-manifest.json

func init() {
	m, err := http2preload.ReadManifest("preload-manifest.json")
	if err != nil {
		panic(err)
	}
	http.Handle("/", m.Handler(handleRoot))
	http.HandleFunc("/gopher", handleGopher)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleGopher(w http.ResponseWriter, r *http.Request) {
	a := &http2preload.Asset{URL: gopher, Type: http2preload.Image}
	s := "http"
	if r.TLS != nil {
		s = "https"
	}
	http2preload.AddHeader(w.Header(), s, r.Host, a)
	fmt.Fprintf(w, `<img src="%s">`, gopher)
}
