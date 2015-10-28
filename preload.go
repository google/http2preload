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

// Package http2preload provides a way to manipulate HTTP/2 preload header.
package http2preload

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

// Request types, as specified by https://fetch.spec.whatwg.org/#concept-request-type
const (
	Audio  = "audio"
	Font   = "font"
	Image  = "image"
	Script = "script"
	Style  = "style"
	Track  = "track"
	Video  = "video"
)

// Asset is an element of Manifest values.
type Asset struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

// Manifest is the push manifest: a collection of assets
// with each subset (map values) mapped to a URL path (map keys).
type Manifest map[string][]*Asset

// Handler creates a new handler which adds preload header(s)
// if in-flight request URL matches one of the m entries.
func (m Manifest) Handler(f http.HandlerFunc) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		if a, ok := m[r.URL.Path]; ok {
			s := r.Header.Get("x-forwarded-proto")
			if s == "" && r.TLS != nil {
				s = "https"
			}
			if s == "" {
				s = "http"
			}
			AddHeader(w.Header(), s, r.Host, a...)
		}
		f(w, r)
	}
	return http.HandlerFunc(h)
}

// AddHeader adds "Link: <url>; preload[; as=xxx]" header to h for each Asset a.
// scheme://base will be prepended to all a.URL which are not prefixed
// with "http:" or "https:".
func AddHeader(hdr http.Header, scheme, base string, a ...*Asset) {
	for _, x := range a {
		xu := x.URL
		if !strings.HasPrefix(xu, "https:") && !strings.HasPrefix(xu, "http:") {
			xu = scheme + "://" + path.Join(base, xu)
		}
		v := fmt.Sprintf("<%s>; rel=preload", xu)
		if x.Type != "" {
			v += "; as=" + x.Type
		}
		hdr.Add("link", v)
		hdr.Add("x-associated-content", fmt.Sprintf("%q", xu))
	}
}

var (
	manifestCacheMu sync.Mutex // guards manifestCache
	manifestCache   = map[string]Manifest{}
)

// ReadManifest reads a push manifest from name file.
// It caches the value in memory so that subsequent requests
// won't hit the disk again.
//
// A manifest file can also be generated with a tool
// found in cmd/http2preload-manifest.
func ReadManifest(name string) (Manifest, error) {
	manifestCacheMu.Lock()
	defer manifestCacheMu.Unlock()
	if m, ok := manifestCache[name]; ok {
		return m, nil
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var m Manifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, err
	}
	manifestCache[name] = m
	return m, nil
}
