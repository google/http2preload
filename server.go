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

package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/http2preload/generate"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/manifest", manifest)
}

func manifest(w http.ResponseWriter, r *http.Request) {
	var markup io.Reader
	ctx := appengine.NewContext(r)
	switch r.Method {
	case "POST":
		markup = r.Body
	case "GET":
		u := r.FormValue("url")
		if !strings.HasPrefix(u, "http:") && !strings.HasPrefix(u, "https:") {
			u = "https://" + u
		}
		res, err := urlfetch.Client(ctx).Get(u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			w.WriteHeader(res.StatusCode)
			return
		}
		markup = res.Body
	default:
		http.Error(w, "Unsupported method "+r.Method, http.StatusBadRequest)
		return
	}

	// TODO: make abs a slice of domains
	abs := false
	assets, err := generate.SearchHTML(markup, abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(b)
}
