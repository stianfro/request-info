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
