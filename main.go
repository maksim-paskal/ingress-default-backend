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
	"bytes"
	"encoding/json"
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

var (
	buildTime string = "development"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Starting %s", buildTime)

	http.HandleFunc("/", handler)
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc(envDefaults("ERROR_GENERATOR_PATH", "/errorGenerator"), errorGenerator)
	err := http.ListenAndServe(envDefaults("LISTEN", ":80"), nil)

	if err != nil {
		log.Panic(err)
	}
}

func envDefaults(name string, defaultValue string) string {
	result := os.Getenv(name)
	if len(result) > 0 {
		return result
	} else {
		return defaultValue
	}
}

type TemplateData struct {
	TemplateName string
	Code         string
	CodeText     string
	RequestID    string
	Format       string
	Time         string
	OriginalURI  string
	Namespace    string
	IngressName  string
	ServiceName  string
	ServicePort  string
}

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

	target_code_int, err := strconv.Atoi(data.Code)

	if err != nil {
		target_code_int = 400
		log.Warnf("%q looks like not a number.\n", data.Code)
	}

	data.CodeText = http.StatusText(target_code_int)

	w.WriteHeader(target_code_int)

	f := sprig.FuncMap()

	data.TemplateName = fmt.Sprintf("templates/%d.html", target_code_int)

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

	js, err := json.Marshal(data)
	if err != nil {
		log.Error(err)
	}
	var logMessage bytes.Buffer

	_, err = logMessage.Write(js)

	if err != nil {
		log.Error(err)
	}
	fmt.Println(logMessage.String())
}

func errorGenerator(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	code := r.Form.Get("code")

	if len(code) == 0 {
		code = "200"
	}

	target_code_int, err := strconv.Atoi(code)

	if err != nil {
		target_code_int = 400
		log.Printf("%q looks like not a number.\n", code)
	}

	w.WriteHeader(target_code_int)
	_, err = w.Write([]byte(fmt.Sprintf("code=%d", target_code_int)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte(buildTime))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
