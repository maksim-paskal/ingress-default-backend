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

//nolint:gosec
import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/maksim-paskal/ingress-default-backend/pkg/web"
	log "github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals
var (
	buildTime  = "dev"
	logLevel   = flag.String("log.level", "INFO", "log level")
	httpListen = flag.String("http.listen", ":8080", "server listen")
)

const (
	httpReadTimeout = 5 * time.Second
)

func main() {
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(level)
	log.SetReportCaller(true)

	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Starting %s...", buildTime)

	log.Infof("Listen on port %s", *httpListen)

	server := &http.Server{
		Addr:        *httpListen,
		Handler:     web.Handlers(),
		ReadTimeout: httpReadTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Panic(err)
	}
}
