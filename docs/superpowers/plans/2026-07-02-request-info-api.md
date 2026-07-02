# Request Info API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a small Go API that returns request metadata as JSON and deploy it to Minato as `acme/request-info`.

**Architecture:** The app is a Go standard library HTTP service. Request metadata lives in `internal/requestinfo`, while `cmd/request-info/main.go` handles process startup, port selection, and server wiring. Development commands are managed through `justfile`, and Minato builds from the public GitHub repo `stianfro/request-info`.

**Tech Stack:** Go 1.22 or newer, Go standard library, `just`, `gh`, Minato MCP tools.

---

## File Structure

- Create `go.mod`: declares module `github.com/stianfro/request-info` and the Go version.
- Create `internal/requestinfo/info.go`: defines response data structures, request extraction, header redaction, stable header sorting, and the HTTP handler.
- Create `internal/requestinfo/info_test.go`: tests request extraction, header redaction, stable headers, and handler JSON responses.
- Create `cmd/request-info/main.go`: reads `PORT`, creates the HTTP server, and starts it.
- Create `cmd/request-info/main_test.go`: tests port resolution.
- Create `justfile`: defines `fmt`, `fmt-check`, `lint`, `test`, `run`, and `ci`.
- Create `README.md`: documents local run and request examples.
- Keep `docs/superpowers/specs/2026-07-02-request-info-api-design.md`: approved design.
- Keep `docs/superpowers/plans/2026-07-02-request-info-api.md`: this plan.

## Task 1: Scaffold module and request metadata package with tests

**Files:**
- Create: `go.mod`
- Create: `internal/requestinfo/info.go`
- Create: `internal/requestinfo/info_test.go`

- [ ] **Step 1: Create `go.mod`**

```go
module github.com/stianfro/request-info

go 1.22
```

- [ ] **Step 2: Write failing request metadata tests**

Create `internal/requestinfo/info_test.go`:

```go
package requestinfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFromRequestCollectsRequestMetadata(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "https://example.com/widgets?color=blue", nil)
	req.Header.Set("X-Test", "one")
	req.Header.Add("X-Test", "two")
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=secret")
	req.RemoteAddr = "203.0.113.10:45678"
	req.RequestURI = "/widgets?color=blue"
	req.ContentLength = 42
	req.TransferEncoding = []string{"chunked"}

	info := FromRequest(req, time.Date(2026, 7, 2, 1, 2, 3, 0, time.UTC))

	if info.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", info.Method, http.MethodPost)
	}
	if info.Path != "/widgets" {
		t.Fatalf("path = %q, want /widgets", info.Path)
	}
	if info.RawQuery != "color=blue" {
		t.Fatalf("raw query = %q, want color=blue", info.RawQuery)
	}
	if info.Host != "example.com" {
		t.Fatalf("host = %q, want example.com", info.Host)
	}
	if info.Protocol != "HTTP/1.1" {
		t.Fatalf("protocol = %q, want HTTP/1.1", info.Protocol)
	}
	if info.RemoteAddr != "203.0.113.10:45678" {
		t.Fatalf("remote addr = %q, want 203.0.113.10:45678", info.RemoteAddr)
	}
	if info.RequestURI != "/widgets?color=blue" {
		t.Fatalf("request uri = %q, want /widgets?color=blue", info.RequestURI)
	}
	if info.ContentLength != 42 {
		t.Fatalf("content length = %d, want 42", info.ContentLength)
	}
	if len(info.TransferEncoding) != 1 || info.TransferEncoding[0] != "chunked" {
		t.Fatalf("transfer encoding = %#v, want [chunked]", info.TransferEncoding)
	}
	if !info.TLSEnabled {
		t.Fatal("tls enabled = false, want true")
	}
	if info.Timestamp != "2026-07-02T01:02:03Z" {
		t.Fatalf("timestamp = %q, want 2026-07-02T01:02:03Z", info.Timestamp)
	}

	wantHeaders := []Header{
		{Name: "Authorization", Values: []string{redactedValue}},
		{Name: "Cookie", Values: []string{redactedValue}},
		{Name: "X-Test", Values: []string{"one", "two"}},
	}
	assertHeaders(t, info.Headers, wantHeaders)
}

func TestHeadersAreSortedByName(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("Z-Last", "z")
	req.Header.Set("A-First", "a")
	req.Header.Set("M-Middle", "m")

	info := FromRequest(req, time.Date(2026, 7, 2, 1, 2, 3, 0, time.UTC))

	wantHeaders := []Header{
		{Name: "A-First", Values: []string{"a"}},
		{Name: "M-Middle", Values: []string{"m"}},
		{Name: "Z-Last", Values: []string{"z"}},
	}
	assertHeaders(t, info.Headers, wantHeaders)
}

func TestHandlerWritesJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/hello?name=codex", nil)
	recorder := httptest.NewRecorder()
	handler := Handler(func() time.Time {
		return time.Date(2026, 7, 2, 1, 2, 3, 0, time.UTC)
	})

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}

	var info Info
	if err := json.Unmarshal(recorder.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if info.Path != "/hello" {
		t.Fatalf("path = %q, want /hello", info.Path)
	}
	if info.RawQuery != "name=codex" {
		t.Fatalf("raw query = %q, want name=codex", info.RawQuery)
	}
}

func assertHeaders(t *testing.T, got []Header, want []Header) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("headers length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name {
			t.Fatalf("headers[%d].name = %q, want %q", i, got[i].Name, want[i].Name)
		}
		if len(got[i].Values) != len(want[i].Values) {
			t.Fatalf("headers[%d].values length = %d, want %d", i, len(got[i].Values), len(want[i].Values))
		}
		for j := range want[i].Values {
			if got[i].Values[j] != want[i].Values[j] {
				t.Fatalf("headers[%d].values[%d] = %q, want %q", i, j, got[i].Values[j], want[i].Values[j])
			}
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail because the package is not implemented**

Run:

```bash
go test ./internal/requestinfo
```

Expected: FAIL with errors such as `undefined: FromRequest`, `undefined: Header`, `undefined: redactedValue`, `undefined: Handler`, and `undefined: Info`.

- [ ] **Step 4: Implement request metadata extraction and handler**

Create `internal/requestinfo/info.go`:

```go
package requestinfo

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

const redactedValue = "[REDACTED]"

type Info struct {
	Method           string   `json:"method"`
	Path             string   `json:"path"`
	RawQuery         string   `json:"raw_query"`
	Host             string   `json:"host"`
	Protocol         string   `json:"protocol"`
	RemoteAddr       string   `json:"remote_addr"`
	RequestURI       string   `json:"request_uri"`
	Headers          []Header `json:"headers"`
	ContentLength    int64    `json:"content_length"`
	TransferEncoding []string `json:"transfer_encoding"`
	TLSEnabled       bool     `json:"tls_enabled"`
	Timestamp        string   `json:"timestamp"`
}

type Header struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

func FromRequest(r *http.Request, now time.Time) Info {
	return Info{
		Method:           r.Method,
		Path:             r.URL.Path,
		RawQuery:         r.URL.RawQuery,
		Host:             r.Host,
		Protocol:         r.Proto,
		RemoteAddr:       r.RemoteAddr,
		RequestURI:       r.RequestURI,
		Headers:          sortedHeaders(r.Header),
		ContentLength:    r.ContentLength,
		TransferEncoding: append([]string(nil), r.TransferEncoding...),
		TLSEnabled:       r.TLS != nil,
		Timestamp:        now.UTC().Format(time.RFC3339),
	}
}

func Handler(now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(FromRequest(r, now())); err != nil {
			slog.Error("encode request info response", "error", err)
		}
	})
}

func sortedHeaders(headers http.Header) []Header {
	result := make([]Header, 0, len(headers))
	for name, values := range headers {
		result = append(result, Header{
			Name:   name,
			Values: redactHeaderValues(name, values),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

func redactHeaderValues(name string, values []string) []string {
	if isSensitiveHeader(name) {
		return []string{redactedValue}
	}
	return append([]string(nil), values...)
}

func isSensitiveHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "cookie", "set-cookie", "proxy-authorization", "x-api-key":
		return true
	default:
		return false
	}
}
```

- [ ] **Step 5: Run package tests to verify they pass**

Run:

```bash
go test ./internal/requestinfo
```

Expected: PASS.

- [ ] **Step 6: Commit task 1**

```bash
git add go.mod internal/requestinfo/info.go internal/requestinfo/info_test.go
git commit -m "feat: add request info handler"
```

## Task 2: Add HTTP server entrypoint and port tests

**Files:**
- Create: `cmd/request-info/main.go`
- Create: `cmd/request-info/main_test.go`

- [ ] **Step 1: Write failing port resolution tests**

Create `cmd/request-info/main_test.go`:

```go
package main

import "testing"

func TestPortFromEnvDefaultsTo8080(t *testing.T) {
	t.Parallel()

	port, err := portFromEnv("")
	if err != nil {
		t.Fatalf("portFromEnv returned error: %v", err)
	}
	if port != "8080" {
		t.Fatalf("port = %q, want 8080", port)
	}
}

func TestPortFromEnvUsesConfiguredPort(t *testing.T) {
	t.Parallel()

	port, err := portFromEnv("9090")
	if err != nil {
		t.Fatalf("portFromEnv returned error: %v", err)
	}
	if port != "9090" {
		t.Fatalf("port = %q, want 9090", port)
	}
}

func TestPortFromEnvRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	_, err := portFromEnv("not-a-port")
	if err == nil {
		t.Fatal("portFromEnv returned nil error, want error")
	}
}

func TestPortFromEnvRejectsOutOfRangePort(t *testing.T) {
	t.Parallel()

	_, err := portFromEnv("70000")
	if err == nil {
		t.Fatal("portFromEnv returned nil error, want error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail because the function is missing**

Run:

```bash
go test ./cmd/request-info
```

Expected: FAIL with `undefined: portFromEnv`.

- [ ] **Step 3: Implement the server entrypoint**

Create `cmd/request-info/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/stianfro/request-info/internal/requestinfo"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	port, err := portFromEnv(os.Getenv("PORT"))
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           requestinfo.Handler(time.Now),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting request-info server", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func portFromEnv(value string) (string, error) {
	if value == "" {
		return "8080", nil
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		return "", fmt.Errorf("invalid PORT %q: %w", value, err)
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid PORT %q: must be between 1 and 65535", value)
	}
	return value, nil
}
```

- [ ] **Step 4: Run entrypoint tests to verify they pass**

Run:

```bash
go test ./cmd/request-info
```

Expected: PASS.

- [ ] **Step 5: Run all Go tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit task 2**

```bash
git add cmd/request-info/main.go cmd/request-info/main_test.go
git commit -m "feat: add request info server"
```

## Task 3: Add development commands and documentation

**Files:**
- Create: `justfile`
- Create: `README.md`

- [ ] **Step 1: Create `justfile`**

```just
set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

fmt:
    gofmt -w cmd internal

fmt-check:
    test -z "$(gofmt -l cmd internal)"

lint:
    go vet ./...

test:
    go test ./...

run:
    go run ./cmd/request-info

ci: fmt-check lint test
```

- [ ] **Step 2: Create `README.md`**

```markdown
# request-info

A small Go API that returns information about the incoming HTTP request as JSON.

## Local development

Run checks:

```bash
just ci
```

Run the server:

```bash
just run
```

Use a custom port:

```bash
PORT=9090 just run
```

Send a request:

```bash
curl -H 'X-Demo: hello' 'http://localhost:8080/example?debug=true'
```

## Response

The API returns JSON with request metadata:

- method
- path
- raw query string
- host
- protocol
- remote address
- request URI
- headers
- content length
- transfer encoding
- TLS enabled flag
- server timestamp

Sensitive headers are redacted before they are returned.
```

- [ ] **Step 3: Run formatting through just**

Run:

```bash
just fmt
```

Expected: command exits 0.

- [ ] **Step 4: Run final local checks**

Run:

```bash
just ci
```

Expected: PASS for formatting check, `go vet ./...`, and `go test ./...`.

- [ ] **Step 5: Commit task 3**

```bash
git add justfile README.md
git commit -m "chore: add development commands"
```

## Task 4: Publish the repository to GitHub

**Files:**
- No source file changes expected.

- [ ] **Step 1: Verify working tree before publishing**

Run:

```bash
git status --short
```

Expected: no output.

- [ ] **Step 2: Create the public GitHub repository and push `main`**

Run:

```bash
gh repo create stianfro/request-info --public --source=. --remote=origin --push
```

Expected: `gh` creates `https://github.com/stianfro/request-info`, adds remote `origin`, and pushes `main`.

If the repository already exists, run:

```bash
git remote add origin https://github.com/stianfro/request-info.git
git push -u origin main
```

Expected: `main` is pushed to `origin`.

- [ ] **Step 3: Verify remote state**

Run:

```bash
git remote -v
git status --short
```

Expected: `origin` points at `https://github.com/stianfro/request-info.git` and `git status --short` has no output.

## Task 5: Deploy to Minato

**Files:**
- No source file changes expected.

- [ ] **Step 1: Check Minato builders**

Use Minato tool `minato_list_builders` with:

```json
{"tenant":"acme"}
```

Expected: the `go` builder is listed and ready.

- [ ] **Step 2: Create the Minato app**

Use Minato tool `minato_create_app` with:

```json
{
  "tenant": "acme",
  "name": "request-info",
  "access": "public",
  "port": 8080,
  "minScale": 0,
  "maxScale": 1
}
```

Expected: app `request-info` exists in tenant `acme` and waits for its first build.

If the app already exists, use Minato tool `minato_get_app` with:

```json
{"tenant":"acme","name":"request-info"}
```

Expected: app details are returned. Continue with the build.

- [ ] **Step 3: Start a build from GitHub**

Use Minato tool `minato_create_build` with:

```json
{
  "tenant": "acme",
  "name": "request-info",
  "url": "https://github.com/stianfro/request-info",
  "ref": "main",
  "builder": "go"
}
```

Expected: a build is queued and the response includes a build ID.

- [ ] **Step 4: Poll the build until it succeeds**

Use Minato tool `minato_get_build_status` with the returned build ID:

```json
{"tenant":"acme","name":"request-info","buildId":"BUILD_ID_FROM_STEP_3"}
```

Expected: repeat until phase is `Succeeded`. If phase is `Failed`, read logs with `minato_get_build_logs`, fix the code, commit, push, and retry with a new build.

- [ ] **Step 5: Deploy the succeeded build**

Use Minato tool `minato_deploy_build` with:

```json
{"tenant":"acme","name":"request-info","buildId":"BUILD_ID_FROM_STEP_3"}
```

Expected: Minato starts rolling out the built image.

- [ ] **Step 6: Poll app status until live**

Use Minato tool `minato_get_app` with:

```json
{"tenant":"acme","name":"request-info"}
```

Expected: app reports a live or ready state. If it reports a failed state, inspect the suggestions in the response, fix the problem, commit, push, rebuild, and redeploy.

- [ ] **Step 7: Record deployment details**

Collect these details for the final response:

- Repository URL: `https://github.com/stianfro/request-info`
- Tenant: `acme`
- App: `request-info`
- Build ID
- Current version
- Live status

## Final Verification

- [ ] Run `just ci` after all code changes and before final handoff.
- [ ] Run `git status --short` after commits and deployment. Expected: no output.
- [ ] Report all commits made.
- [ ] Report Minato deployment status and build ID.
