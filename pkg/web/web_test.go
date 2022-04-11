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
	"strconv"
	"testing"

	"github.com/maksim-paskal/ingress-default-backend/pkg/web"
)

// nolint:gochecknoglobals
var (
	client = &http.Client{}
	ts     = httptest.NewServer(web.Handlers())
	ctx    = context.Background()
)

func TestHealtz(t *testing.T) {
	t.Parallel()

	url := fmt.Sprintf("%s/healthz", ts.URL)

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

	if err := flag.Set("templates", "testdata"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Code", "504")
	req.Header.Set("X-Request-ID", "1")
	req.Header.Set("X-Format", "2")
	req.Header.Set("X-Original-URI", "3")
	req.Header.Set("X-Namespace", "4")
	req.Header.Set("X-Ingress-Name", "5")
	req.Header.Set("X-Service-Name", "6")
	req.Header.Set("X-Service-Port", "7")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 504 {
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
