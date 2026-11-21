package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newEchoServer(t *testing.T, rec *recorder) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.add("method=" + r.Method)
		rec.add("body=" + string(body))
		rec.add("ctype=" + r.Header.Get("Content-Type"))
		rec.add("x-test=" + r.Header.Get("X-Test"))
		if r.URL.Path == "/missing" {
			http.Error(w, "nope", http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, "hello from server")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHTTPMethods(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantMethod string
		wantBody   string
	}{
		{"get", []string{"http", "get"}, "GET", ""},
		{"delete", []string{"http", "delete"}, "DELETE", ""},
		{"post", []string{"http", "post", "-d", `{"a":1}`}, "POST", `{"a":1}`},
		{"put", []string{"http", "put", "-d", "payload"}, "PUT", "payload"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			srv := newEchoServer(t, rec)
			out, err := runCLI(t, append(tt.args, srv.URL)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			mustContain(t, out, "Status: 200 OK", "hello from server")
			mustContain(t, rec.all(), "method="+tt.wantMethod, "body="+tt.wantBody)
		})
	}
}

func TestHTTPHeaders(t *testing.T) {
	rec := &recorder{}
	srv := newEchoServer(t, rec)
	if _, err := runCLI(t, "http", "get", srv.URL, "-H", "X-Test: abc"); err != nil {
		t.Fatal(err)
	}
	mustContain(t, rec.all(), "x-test=abc")
}

func TestHTTPDefaultContentTypeForBody(t *testing.T) {
	rec := &recorder{}
	srv := newEchoServer(t, rec)
	if _, err := runCLI(t, "http", "post", srv.URL, "-d", "{}"); err != nil {
		t.Fatal(err)
	}
	mustContain(t, rec.all(), "ctype=application/json")
}

func TestHTTPCustomContentTypeNotOverridden(t *testing.T) {
	rec := &recorder{}
	srv := newEchoServer(t, rec)
	if _, err := runCLI(t, "http", "post", srv.URL, "-d", "a=b", "-H", "Content-Type: text/plain"); err != nil {
		t.Fatal(err)
	}
	mustContain(t, rec.all(), "ctype=text/plain")
}

func TestHTTPNon2xxStillPrintsStatus(t *testing.T) {
	srv := newEchoServer(t, &recorder{})
	out, err := runCLI(t, "http", "get", srv.URL+"/missing")
	if err != nil {
		t.Fatalf("4xx should not be a CLI error: %v", err)
	}
	mustContain(t, out, "Status: 404 Not Found", "nope")
}

func TestHTTPInvalidHeader(t *testing.T) {
	srv := newEchoServer(t, &recorder{})
	_, err := runCLI(t, "http", "get", srv.URL, "-H", "NoColonHere")
	if err == nil {
		t.Fatal("expected error for malformed header")
	}
	mustContain(t, err.Error(), "invalid header format")
}

func TestHTTPRequiresURL(t *testing.T) {
	if _, err := runCLI(t, "http", "get"); err == nil {
		t.Fatal("expected error when url missing")
	}
}

func TestHTTPUnreachable(t *testing.T) {
	_, err := runCLI(t, "http", "get", "http://127.0.0.1:1", "-t", "1s")
	if err == nil {
		t.Fatal("expected connection error")
	}
	mustContain(t, err.Error(), "request failed")
}

func TestHTTPSHasAllVerbs(t *testing.T) {
	// An explicit scheme must not be rewritten, so a plain-http test server
	// lets us exercise every https verb without TLS certificates.
	tests := []struct {
		name       string
		args       []string
		wantMethod string
		wantBody   string
	}{
		{"get", []string{"https", "get"}, "GET", ""},
		{"delete", []string{"https", "delete"}, "DELETE", ""},
		{"post", []string{"https", "post", "-d", "x=1"}, "POST", "x=1"},
		{"put", []string{"https", "put", "-d", "y=2"}, "PUT", "y=2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			srv := newEchoServer(t, rec)
			out, err := runCLI(t, append(tt.args, srv.URL, "-H", "X-Test: s")...)
			if err != nil {
				t.Fatal(err)
			}
			mustContain(t, out, "Status: 200 OK")
			mustContain(t, rec.all(), "method="+tt.wantMethod, "body="+tt.wantBody, "x-test=s")
		})
	}
}

func TestHTTPSRejectsUntrustedCertificate(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	_, err := runCLI(t, "https", "get", srv.URL)
	if err == nil {
		t.Fatal("expected certificate verification error")
	}
	mustContain(t, err.Error(), "request failed")
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct{ in, scheme, want string }{
		{"example.com/a", "https", "https://example.com/a"},
		{"localhost:8080", "http", "http://localhost:8080"},
		{"http://x.test", "https", "http://x.test"},
		{"https://x.test", "http", "https://x.test"},
	}
	for _, tt := range tests {
		if got := normalizeURL(tt.in, tt.scheme); got != tt.want {
			t.Errorf("normalizeURL(%q,%q) = %q, want %q", tt.in, tt.scheme, got, tt.want)
		}
	}
}
