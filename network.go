package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func isProbe(name string) bool {
	return contains([]string{"ping", "traceroute", "dns", "tcp", "http", "tls"}, name)
}
func (a *App) probe(parent context.Context, name string, args []string) Result {
	ctx, cancel := context.WithTimeout(parent, time.Duration(a.Config.TimeoutSeconds)*time.Second)
	defer cancel()
	r := result(name, OK, "Go network", "Probe succeeded", nil)
	fail := func(e error) Result { r.Status = Unknown; r.Summary = e.Error(); r.Raw = e.Error(); return r }
	switch name {
	case "ping", "traceroute":
		if e := requireArgs(args, 1); e != nil {
			return fail(e)
		}
		if runtime.GOOS != "windows" {
			return fail(fmt.Errorf("this probe uses Windows inbox tools"))
		}
		host := args[0]
		if host == "" || strings.HasPrefix(host, "-") || strings.ContainsAny(host, "/\\ \t\r\n") {
			return fail(fmt.Errorf("invalid probe host"))
		}
		command := "ping.exe"
		toolArgs := []string{"-n", "4", "-w", "1000", host}
		if name == "traceroute" {
			command = "tracert.exe"
			toolArgs = []string{"-d", "-h", "15", "-w", "1000", host}
		}
		b, stderr, e := runBounded(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", command), toolArgs, os.Environ())
		r.Source = "Windows " + command
		r.Raw = string(b) + stderr
		r.Data = string(b)
		if e != nil {
			return fail(fmt.Errorf("probe failed: %w; %s", e, strings.TrimSpace(string(b))))
		}
		r.Limitations = []string{"Localized tool output preserved; traceroute completion does not establish end-to-end reachability"}
	case "tcp":
		if e := requireArgs(args, 2); e != nil {
			return fail(e)
		}
		port, e := strconv.Atoi(args[1])
		if e != nil || port < 1 || port > 65535 {
			return fail(fmt.Errorf("TCP port must be 1..65535"))
		}
		start := time.Now()
		conn, e := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(args[0], args[1]))
		if e != nil {
			return fail(e)
		}
		defer conn.Close()
		r.Data = map[string]any{"Host": args[0], "Port": port, "LatencyMs": float64(time.Since(start).Microseconds()) / 1000, "Local": conn.LocalAddr().String(), "Remote": conn.RemoteAddr().String()}
		r.Limitations = []string{"TCP handshake only; application health was not checked"}
	case "dns":
		if len(args) != 1 && len(args) != 3 {
			return fail(fmt.Errorf("dns NAME [--server IP[:PORT]]"))
		}
		resolver := net.DefaultResolver
		server := "system resolver"
		if len(args) == 3 {
			if args[1] != "--server" {
				return fail(fmt.Errorf("expected --server"))
			}
			server = args[2]
			if _, _, e := net.SplitHostPort(server); e != nil {
				server = net.JoinHostPort(strings.Trim(server, "[]"), "53")
			}
			host, port, e := net.SplitHostPort(server)
			if e != nil || net.ParseIP(host) == nil {
				return fail(fmt.Errorf("DNS server must be an IP address with optional port"))
			}
			n, e := strconv.Atoi(port)
			if e != nil || n < 1 || n > 65535 {
				return fail(fmt.Errorf("invalid DNS server port"))
			}
			resolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server)
			}}
		}
		ips, e := resolver.LookupHost(ctx, args[0])
		if e != nil {
			return fail(e)
		}
		r.Data = map[string]any{"Name": args[0], "Resolver": server, "Addresses": ips}
		r.Limitations = []string{"A/AAAA lookup only; results do not establish application reachability"}
	case "http":
		if e := requireArgs(args, 1); e != nil {
			return fail(e)
		}
		u, e := url.Parse(args[0])
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return fail(fmt.Errorf("use an http(s) URL without embedded credentials"))
		}
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, args[0], nil)
		if e != nil {
			return fail(e)
		}
		transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: time.Duration(a.Config.TimeoutSeconds) * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second}
		defer transport.CloseIdleConnections()
		client := &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		start := time.Now()
		response, e := client.Do(req)
		if e != nil {
			return fail(e)
		}
		defer response.Body.Close()
		r.Data = map[string]any{"URL": args[0], "StatusCode": response.StatusCode, "Status": response.Status, "LatencyMs": time.Since(start).Milliseconds(), "Headers": response.Header}
		if response.StatusCode >= 500 {
			r.Status = Critical
		} else if response.StatusCode >= 400 {
			r.Status = Warning
		}
		r.Limitations = []string{"GET request; redirects are not followed; uses environment proxies, not Windows WinHTTP/WinINET proxy settings; response body not retained"}
	case "tls":
		if len(args) < 1 || len(args) > 2 {
			return fail(fmt.Errorf("tls HOST [PORT]"))
		}
		port := "443"
		if len(args) == 2 {
			port = args[1]
		}
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return fail(fmt.Errorf("invalid TLS port"))
		}
		host := args[0]
		// Inspection handshake captures the chain; trust and hostname verification are
		// mandatory below and determine the result. No application data is sent.
		d := tls.Dialer{NetDialer: &net.Dialer{}, Config: &tls.Config{ServerName: host, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}
		conn, e := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
		if e != nil {
			return fail(e)
		}
		defer conn.Close()
		state := conn.(*tls.Conn).ConnectionState()
		if len(state.PeerCertificates) == 0 {
			return fail(fmt.Errorf("peer returned no certificate"))
		}
		leaf := state.PeerCertificates[0]
		roots, e := x509.SystemCertPool()
		if e != nil {
			return fail(e)
		}
		intermediate := x509.NewCertPool()
		for _, c := range state.PeerCertificates[1:] {
			intermediate.AddCert(c)
		}
		_, verifyErr := leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: roots, Intermediates: intermediate})
		validation := "valid"
		if verifyErr != nil {
			r.Status = Critical
			validation = verifyErr.Error()
			r.Summary = "TLS certificate validation failed"
		} else if time.Until(leaf.NotAfter) < 30*24*time.Hour {
			r.Status = Warning
			r.Summary = "TLS certificate expires within 30 days"
		}
		chain := []map[string]any{}
		for _, c := range state.PeerCertificates {
			chain = append(chain, map[string]any{"Subject": c.Subject.String(), "Issuer": c.Issuer.String(), "DNSNames": c.DNSNames, "NotBefore": c.NotBefore, "NotAfter": c.NotAfter, "Serial": c.SerialNumber.String()})
		}
		r.Data = map[string]any{"Host": host, "Port": port, "Version": tls.VersionName(state.Version), "Cipher": tls.CipherSuiteName(state.CipherSuite), "Validation": validation, "Chain": chain}
		r.Limitations = []string{"Certificate validity, hostname and system trust checked; no online revocation/OCSP check; only TLS 1.2+ offered"}
	}
	if r.Raw == "" {
		b, _ := json.Marshal(r.Data)
		r.Raw = string(b)
	}
	return r
}
