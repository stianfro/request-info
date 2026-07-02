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
