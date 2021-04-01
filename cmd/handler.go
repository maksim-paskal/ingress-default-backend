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
package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Masterminds/sprig"
	log "github.com/sirupsen/logrus"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// https://kubernetes.github.io/ingress-nginx/user-guide/custom-errors/
	data := TemplateData{
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

	data.CodeText = http.StatusText(targetCodeInt)

	w.WriteHeader(targetCodeInt)

	f := sprig.FuncMap()

	data.TemplateName = fmt.Sprintf("templates/%d.html", targetCodeInt)

	if _, err := os.Stat(data.TemplateName); err != nil {
		data.TemplateName = "templates/default.html"
	}

	tmpl := template.New(filepath.Base(data.TemplateName))
	tmpl, err = tmpl.Funcs(f).ParseFiles(data.TemplateName)

	if err != nil {
		log.Error(err)
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		log.Error(err)
	}

	log.WithFields(data.Fields()).Info()
}
