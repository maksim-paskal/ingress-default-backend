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
package web_test

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/maksim-paskal/ingress-default-backend/pkg/web"
)

const xCode = "X-Code"

//nolint:gochecknoglobals
var (
	client = &http.Client{}
	ts     = httptest.NewServer(web.Handlers())
	ctx    = context.Background()
)

func TestMain(m *testing.M) {
	// set before any request, the flag is read by handlers
	if err := flag.Set("templates", "testdata"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// get sends request with headers and returns status code and body.
func get(t *testing.T, path string, headers map[string]string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return resp.StatusCode, string(body)
}

func TestHealtz(t *testing.T) {
	t.Parallel()

	url := ts.URL + "/healthz"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("status code is not 200")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Log(string(body))

	if m := "ok"; string(body) != m {
		t.Fatal("not correct response")
	}
}

func TestErrorGenerator(t *testing.T) {
	t.Parallel()

	const code = "402"

	codeInt, err := strconv.Atoi(code)
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("%s/errorGenerator?code=%s", ts.URL, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != codeInt {
		t.Fatalf("status code is not %d", codeInt)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Log(string(body))

	if m := "code=" + code; string(body) != m {
		t.Fatal("not correct response")
	}
}

func TestTemplate(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Code", "504")
	req.Header.Set("X-Request-ID", "1")
	req.Header.Set("X-Format", "2")
	req.Header.Set("X-Original-URI", "3") //nolint:canonicalheader
	req.Header.Set("X-Namespace", "4")
	req.Header.Set("X-Ingress-Name", "5")
	req.Header.Set("X-Service-Name", "6")
	req.Header.Set("X-Service-Port", "7")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status code %d is not correct", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Log(string(body))

	const correctBody = `Code=504
RequestID=1
Format=2
OriginalURI=3
Namespace=4
IngressName=5
ServiceName=6
ServicePort=7`

	if string(body) != correctBody {
		t.Fatalf("not correct response [%s],[%s]", string(body), correctBody)
	}
}

func TestErrorGeneratorInvalidCode(t *testing.T) {
	t.Parallel()

	// out of range codes must not panic WriteHeader (connection reset)
	for _, code := range []string{"0", "-1", "100", "199", "600", "1000", "99999"} {
		status, body := get(t, "/errorGenerator?code="+code, nil)

		if status != http.StatusBadRequest {
			t.Errorf("code=%s: status %d, want 400, body %q", code, status, body)
		}
	}

	// not a number
	status, body := get(t, "/errorGenerator?code=abc", nil)
	if status != http.StatusBadRequest || body != "code=400" {
		t.Errorf("code=abc: status %d body %q", status, body)
	}
}

func TestErrorGeneratorBadForm(t *testing.T) {
	t.Parallel()

	status, body := get(t, "/errorGenerator?code=%zz", nil)

	if status != http.StatusBadRequest {
		t.Errorf("status %d, want 400", status)
	}

	if strings.Contains(body, "code=200") {
		t.Errorf("error response has success payload: %q", body)
	}
}

func TestErrorGeneratorNoBody(t *testing.T) {
	t.Parallel()

	for _, code := range []int{http.StatusNoContent, http.StatusNotModified} {
		status, body := get(t, fmt.Sprintf("/errorGenerator?code=%d", code), nil)

		if status != code || body != "" {
			t.Errorf("code %d: status %d body %q", code, status, body)
		}
	}
}

func TestHandlerInvalidCode(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"0", "100", "101", "199", "600", "1000"} {
		status, _ := get(t, "/", map[string]string{xCode: code})

		if status != http.StatusInternalServerError {
			t.Errorf("X-Code=%s: status %d, want 500", code, status)
		}
	}
}

func TestHandlerNoBody(t *testing.T) {
	t.Parallel()

	for _, code := range []int{http.StatusNoContent, http.StatusNotModified} {
		status, body := get(t, "/", map[string]string{xCode: strconv.Itoa(code)})

		if status != code || body != "" {
			t.Errorf("X-Code=%d: status %d body %q", code, status, body)
		}
	}
}

func TestHandlerMissingTemplate(t *testing.T) {
	t.Parallel()

	// testdata has neither 404.html nor default.html
	status, body := get(t, "/", map[string]string{xCode: "404"})

	if status != http.StatusInternalServerError {
		t.Errorf("status %d, want 500", status)
	}

	if strings.Contains(body, "testdata") || strings.Contains(body, "no such file") {
		t.Errorf("response leaks internal details: %q", body)
	}
}

func TestHandlerNotNumericCode(t *testing.T) {
	t.Parallel()

	// the rendered code must be the same as the status code
	for _, code := range []string{"abc", ""} {
		status, body := get(t, "/", map[string]string{xCode: code})

		if status != http.StatusBadRequest || body != "Code=400" {
			t.Errorf("X-Code=%q: status %d body %q", code, status, body)
		}
	}

	status, body := get(t, "/", map[string]string{xCode: "0400"})
	if status != http.StatusBadRequest || body != "Code=400" {
		t.Errorf("X-Code=0400: status %d body %q", status, body)
	}
}

// not parallel: changes the global templates flag while no other test is running.
func TestTemplateReload(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	file := dir + "/500.html"

	old := flag.Lookup("templates").Value.String()

	if err := flag.Set("templates", dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = flag.Set("templates", old)
	})

	for _, content := range []string{"first", "second version"} {
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		status, body := get(t, "/", map[string]string{xCode: "500"})

		if status != http.StatusInternalServerError || body != content {
			t.Fatalf("want %q, got status %d body %q", content, status, body)
		}
	}
}

func TestTemplateFallback(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()

	old := flag.Lookup("templates").Value.String()

	if err := flag.Set("templates", dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = flag.Set("templates", old)
	})

	files := map[string]string{
		"503.html":     "{{ broken",
		"default.html": "default",
	}

	for name, content := range files {
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// broken template for the code must not hide the real status
	status, body := get(t, "/", map[string]string{xCode: "503"})

	if status != http.StatusServiceUnavailable || body != "default" {
		t.Fatalf("status %d body %q", status, body)
	}
}

// not parallel: changes the global shutdown flag, reset on cleanup.
func TestHealthzShuttingDown(t *testing.T) { //nolint:paralleltest
	web.SetShuttingDown(true)

	t.Cleanup(func() {
		web.SetShuttingDown(false)
	})

	status, _ := get(t, "/healthz", nil)

	if status != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503 while shutting down", status)
	}
}
