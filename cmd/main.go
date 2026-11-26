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
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maksim-paskal/ingress-default-backend/pkg/web"
	log "github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals
var (
	buildTime       = "dev"
	logLevel        = flag.String("log.level", "INFO", "log level")
	httpListen      = flag.String("http.listen", ":8080", "server listen")
	shutdownDelay   = flag.Duration("shutdown.delay", 0, "keep serving after SIGTERM, before closing the listener")
	shutdownTimeout = flag.Duration("shutdown.timeout", 10*time.Second, "max time to wait for in-flight requests")
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", *httpListen)
	if err != nil {
		log.WithError(err).Fatal()
	}

	server := &http.Server{
		Handler:     web.Handlers(),
		ReadTimeout: httpReadTimeout,
	}

	if err := run(ctx, server, listener, *shutdownDelay, *shutdownTimeout); err != nil {
		log.WithError(err).Fatal()
	}

	log.Info("Server stopped")
}

// run serves until ctx is done, then fails /healthz, waits delay,
// and drains in-flight requests for at most timeout.
func run(
	ctx context.Context,
	server *http.Server,
	listener net.Listener,
	delay time.Duration,
	timeout time.Duration,
) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		// the server stopped on its own, it did not get a signal
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	web.SetShuttingDown(true)
	log.Info("Shutting down")

	if delay > 0 {
		log.Infof("Keep serving for %s before closing the listener", delay)
		time.Sleep(delay)
	}

	// ctx is already canceled here, keep its values but not the cancellation
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
