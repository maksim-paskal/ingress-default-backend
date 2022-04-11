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
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/maksim-paskal/ingress-default-backend/pkg/types"
	log "github.com/sirupsen/logrus"
)

var templateFolder = flag.String("templates", "templates", "folder with templates")

func Handlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler)
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/errorGenerator", errorGenerator)

	return mux
}

func handler(w http.ResponseWriter, r *http.Request) { // nolint:funlen
	// https://kubernetes.github.io/ingress-nginx/user-guide/custom-errors/
	data := types.TemplateData{
		Code:        r.Header.Get("X-Code"),
		RequestID:   r.Header.Get("X-Request-ID"),
		Format:      r.Header.Get("X-Format"),
		OriginalURI: r.Header.Get("X-Original-URI"),
		Namespace:   r.Header.Get("X-Namespace"),
		IngressName: r.Header.Get("X-Ingress-Name"),
		ServiceName: r.Header.Get("X-Service-Name"),
		ServicePort: r.Header.Get("X-Service-Port"),
		Time:        time.Now().UTC().Format(time.RFC3339),
	}

	if len(data.Code) == 0 {
		data.Code = "400"
	}

	targetCodeInt, err := strconv.Atoi(data.Code)
	if err != nil {
		targetCodeInt = 400

		log.Warnf("%q looks like not a number.\n", data.Code)
	}

	if targetCodeInt < 100 || targetCodeInt > 999 {
		http.Error(w, "code is not in range 100-999", http.StatusInternalServerError)

		return
	}

	data.CodeText = http.StatusText(targetCodeInt)

	w.WriteHeader(targetCodeInt)

	f := sprig.FuncMap()

	data.TemplateName = fmt.Sprintf("%s/%d.html", *templateFolder, targetCodeInt)

	if _, err := os.Stat(data.TemplateName); err != nil {
		data.TemplateName = fmt.Sprintf("%s/default.html", *templateFolder)
	}

	tmpl := template.New(filepath.Base(data.TemplateName))
	tmpl, err = tmpl.Funcs(f).ParseFiles(data.TemplateName)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.WithError(err).Error()

		return
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.WithError(err).Error()

		return
	}

	log.WithFields(data.Fields()).Info()
}

func healthz(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("ok")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// function that will show template on code.
func errorGenerator(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	code := r.Form.Get("code")

	if len(code) == 0 {
		code = "200"
	}

	targetCodeInt, err := strconv.Atoi(code)
	if err != nil {
		targetCodeInt = 400

		log.Printf("%q looks like not a number.\n", code)
	}

	w.WriteHeader(targetCodeInt)
	_, err = w.Write([]byte(fmt.Sprintf("code=%d", targetCodeInt)))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
