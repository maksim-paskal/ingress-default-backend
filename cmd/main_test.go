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
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

const testWait = 5 * time.Second

type response struct {
	status int
	body   string
	err    error
}

// startServer runs server with run() and returns its URL and the run error channel.
func startServer(
	ctx context.Context,
	t *testing.T,
	handler http.HandlerFunc,
	timeout time.Duration,
) (string, <-chan error) {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: testWait,
	}
	runErr := make(chan error, 1)

	go func() {
		runErr <- run(ctx, server, listener, 0, timeout)
	}()

	return "http://" + listener.Addr().String(), runErr
}

// get sends a request in the background and returns the channel with the result.
func get(url string) <-chan response {
	resCh := make(chan response, 1)

	go func() {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			resCh <- response{err: err}

			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resCh <- response{err: err}

			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		resCh <- response{status: resp.StatusCode, body: string(body), err: err}
	}()

	return resCh
}

// wait fails the test if ch does not receive in time.
func wait[T any](t *testing.T, ch <-chan T, what string) T { //nolint:ireturn // returns the received value
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(testWait):
		t.Fatalf("timeout waiting for %s", what)
	}

	panic("unreachable")
}

func TestRunCompletesInFlightRequest(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url, runErr := startServer(ctx, t, func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(300 * time.Millisecond)

		_, _ = w.Write([]byte("done"))
	}, 5*time.Second)

	resCh := get(url)

	wait(t, started, "request to reach the handler")

	// shutdown starts while the request is in flight
	cancel()

	res := wait(t, resCh, "in-flight request")
	if res.err != nil {
		t.Fatalf("in-flight request failed: %v", res.err)
	}

	if res.status != http.StatusOK || res.body != "done" {
		t.Fatalf("in-flight request: status %d body %q", res.status, res.body)
	}

	if err := wait(t, runErr, "run to return"); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunShutdownTimeout(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// release the stuck handler when the test ends, so the goroutine does not leak
	defer close(release)

	url, runErr := startServer(ctx, t, func(_ http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
	}, 100*time.Millisecond)

	get(url)

	wait(t, started, "request to reach the handler")

	cancel()

	if err := wait(t, runErr, "run to return"); err == nil {
		t.Fatal("run returned nil, want a shutdown timeout error")
	}
}
