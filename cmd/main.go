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
	"flag"
	"net/http"

	// nolint:gosec
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals
var (
	buildTime             = "dev"
	logLevel              = flag.String("log.level", "INFO", "log level")
	httpListen            = flag.String("http.listen", ":80", "server listen")
	exposeErrorGenerator  = flag.Bool("exposeErrorGenerator", false, "expose error generator endpoint")
	errorGeneratorPattern = flag.String("errorGeneratorPattern", "/errorGenerator", "server listen")
)

func main() {
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)

	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Starting %s...", buildTime)

	http.HandleFunc("/", handler)
	http.HandleFunc("/healthz", healthz)

	if *exposeErrorGenerator {
		http.HandleFunc(*errorGeneratorPattern, errorGenerator)
	}

	log.Infof("Listen on port %s", *httpListen)

	err = http.ListenAndServe(*httpListen, nil)
	if err != nil {
		log.Panic(err)
	}
}
