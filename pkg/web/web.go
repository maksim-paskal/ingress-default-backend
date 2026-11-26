/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package web

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/maksim-paskal/ingress-default-backend/pkg/types"
	log "github.com/sirupsen/logrus"
)

const (
	minCode = 200
	maxCode = 599
)

var (
	templateFolder = flag.String("templates", "templates", "folder with templates")
	// built once, html/template copies the entries on Funcs.
	funcMap       = sprig.FuncMap()
	templateCache = make(map[string]cachedTemplate)
	templateMu    sync.RWMutex
)

type cachedTemplate struct {
	modTime time.Time
	size    int64
	tmpl    *template.Template
}

func Handlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler)
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/errorGenerator", errorGenerator)

	return mux
}

// parseCode converts raw to a HTTP status code, not a number is a 400.
func parseCode(raw string) int {
	code, err := strconv.Atoi(raw)
	if err != nil {
		log.Warnf("%q looks like not a number.", raw)

		return http.StatusBadRequest
	}

	return code
}

func validCode(code int) bool {
	return code >= minCode && code <= maxCode
}

// 204 and 304 responses can not have a body.
func bodyAllowed(code int) bool {
	return code != http.StatusNoContent && code != http.StatusNotModified
}

// loadTemplate returns parsed template, parsed templates are cached until the file changes.
func loadTemplate(name string, info os.FileInfo) (*template.Template, error) {
	templateMu.RLock()

	cached, ok := templateCache[name]

	templateMu.RUnlock()

	if ok && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {
		return cached.tmpl, nil
	}

	tmpl, err := template.New(filepath.Base(name)).Funcs(funcMap).ParseFiles(name)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	templateMu.Lock()
	templateCache[name] = cachedTemplate{modTime: info.ModTime(), size: info.Size(), tmpl: tmpl}
	templateMu.Unlock()

	return tmpl, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	// https://kubernetes.github.io/ingress-nginx/user-guide/custom-errors/
	data := types.TemplateData{
		Code:        r.Header.Get("X-Code"),
		RequestID:   r.Header.Get("X-Request-ID"),
		Format:      r.Header.Get("X-Format"),
		OriginalURI: r.Header.Get("X-Original-URI"), //nolint:canonicalheader
		Namespace:   r.Header.Get("X-Namespace"),
		IngressName: r.Header.Get("X-Ingress-Name"),
		ServiceName: r.Header.Get("X-Service-Name"),
		ServicePort: r.Header.Get("X-Service-Port"),
		Time:        time.Now().UTC().Format(time.RFC3339),
	}

	if len(data.Code) == 0 {
		data.Code = strconv.Itoa(http.StatusBadRequest)
	}

	targetCodeInt := parseCode(data.Code)

	if !validCode(targetCodeInt) {
		log.WithFields(data.Fields()).Warnf("code is not in range %d-%d", minCode, maxCode)
		http.Error(w, fmt.Sprintf("code is not in range %d-%d", minCode, maxCode), http.StatusInternalServerError)

		return
	}

	// show the effective code, not the raw header value
	data.Code = strconv.Itoa(targetCodeInt)
	data.CodeText = http.StatusText(targetCodeInt)

	if !bodyAllowed(targetCodeInt) {
		w.WriteHeader(targetCodeInt)
		log.WithFields(data.Fields()).Info()

		return
	}

	// render before writing the status, so a broken template can return 500
	var buf bytes.Buffer

	if err := render(&buf, &data, targetCodeInt); err != nil {
		log.WithError(err).WithField("template", data.TemplateName).Error()
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(targetCodeInt)

	if _, err := buf.WriteTo(w); err != nil {
		log.WithError(err).Error()

		return
	}

	log.WithFields(data.Fields()).Info()
}

// render executes the template for code, falls back to default.html
// when there is no template for the code or it is broken.
func render(buf *bytes.Buffer, data *types.TemplateData, code int) error {
	var err error

	for _, name := range []string{
		fmt.Sprintf("%s/%d.html", *templateFolder, code),
		*templateFolder + "/default.html",
	} {
		buf.Reset()

		data.TemplateName = name

		err = renderFile(buf, data, name)
		if err == nil {
			return nil
		}

		// missing template for the code is normal
		if !errors.Is(err, fs.ErrNotExist) {
			log.WithError(err).WithField("template", name).Error()
		}
	}

	return err
}

func renderFile(buf *bytes.Buffer, data *types.TemplateData, name string) error {
	info, err := os.Stat(name)
	if err != nil {
		return err //nolint:wrapcheck
	}

	tmpl, err := loadTemplate(name, info)
	if err != nil {
		return err
	}

	return tmpl.Execute(buf, data) //nolint:wrapcheck
}

// shuttingDown is set when the server is stopping, /healthz then fails
// so the pod is removed from the service endpoints.
var shuttingDown atomic.Bool

// SetShuttingDown marks the server as shutting down.
func SetShuttingDown(v bool) {
	shuttingDown.Store(v)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	if shuttingDown.Load() {
		http.Error(w, "shutting down", http.StatusServiceUnavailable)

		return
	}

	if _, err := w.Write([]byte("ok")); err != nil {
		log.WithError(err).Error()
	}
}

// function that will show template on code.
func errorGenerator(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	code := r.Form.Get("code")

	if len(code) == 0 {
		code = strconv.Itoa(http.StatusOK)
	}

	targetCodeInt := parseCode(code)

	if !validCode(targetCodeInt) {
		http.Error(w, fmt.Sprintf("code is not in range %d-%d", minCode, maxCode), http.StatusBadRequest)

		return
	}

	w.WriteHeader(targetCodeInt)

	if !bodyAllowed(targetCodeInt) {
		return
	}

	_, err = fmt.Fprintf(w, "code=%d", targetCodeInt) //nolint:gosec // integer only, no user markup
	if err != nil {
		log.WithError(err).Error()
	}
}
