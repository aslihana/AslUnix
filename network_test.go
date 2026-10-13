package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPStatusAndNoRedirect(t *testing.T) {
	for _, tc := range []struct {
		code int
		want Status
	}{{200, OK}, {302, OK}, {404, Warning}, {503, Critical}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://127.0.0.1:1/unwanted")
			w.WriteHeader(tc.code)
		}))
		r := testApp(fakeRunner{}).probe(context.Background(), "http", []string{s.URL})
		s.Close()
		if r.Status != tc.want {
			t.Fatalf("%d => %+v", tc.code, r)
		}
	}
}
func TestHTTPRejectsUntrustedCertificate(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer s.Close()
	r := testApp(fakeRunner{}).probe(context.Background(), "http", []string{s.URL})
	if r.Status != Unknown {
		t.Fatal("untrusted HTTPS succeeded", r)
	}
}
func TestTLSInspectionPreservesUntrustedChain(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer s.Close()
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(s.URL, "https://"))
	r := testApp(fakeRunner{}).probe(context.Background(), "tls", []string{host, port})
	if r.Status != Critical || r.Data == nil {
		t.Fatal(r)
	}
}
func TestTCPProbe(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	r := testApp(fakeRunner{}).probe(context.Background(), "tcp", []string{host, port})
	if r.Status != OK {
		t.Fatal(r)
	}
	if r = testApp(fakeRunner{}).probe(context.Background(), "tcp", []string{host, "65536"}); r.Status != Unknown {
		t.Fatal("invalid port accepted")
	}
}
