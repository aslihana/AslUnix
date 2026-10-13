package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Report struct {
	Version string    `json:"version"`
	Host    string    `json:"host"`
	Time    time.Time `json:"generated_at"`
	Status  Status    `json:"status"`
	Results []Result  `json:"results"`
}

func (a *App) report(ctx context.Context, args []string, input *bufio.Reader, out io.Writer) int {
	dir := a.Config.ReportDirectory
	html := false
	prune := false
	dry := false
	yes := false
	names := healthChecks
	if len(args) > 0 && args[0] == "prune" {
		prune = true
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out", "--checks":
			if i+1 >= len(args) {
				return usageError(out, args[i]+" requires a value")
			}
			key := args[i]
			i++
			if key == "--out" {
				dir = args[i]
			} else {
				names = strings.Split(args[i], ",")
			}
		case "--html":
			html = true
		case "--dry-run":
			dry = true
		case "--yes":
			yes = true
		default:
			return usageError(out, "unsupported report option "+args[i])
		}
	}
	if prune {
		return a.pruneReports(dir, dry, yes, input, out)
	}
	if yes {
		return usageError(out, "--yes is only supported by report prune")
	}
	for _, name := range names {
		if name != "raid" && !collectorNames[name] {
			return usageError(out, "unsupported report check "+name)
		}
		if contains([]string{"permissions", "signature", "vmware", "sql"}, name) {
			return usageError(out, "report check requires arguments: "+name)
		}
	}
	if dry {
		fmt.Fprintln(out, "Would collect", strings.Join(names, ","), "and create text/JSON reports in", dir, "HTML:", html)
		return 0
	}
	rs := []Result{}
	for _, name := range names {
		if name == "raid" {
			rs = append(rs, a.integration(ctx, name, nil))
		} else {
			p, _ := collectorOptions(name, nil)
			rs = append(rs, a.collect(ctx, name, p))
		}
	}
	host, _ := os.Hostname()
	doc := Report{version, host, time.Now().UTC(), worst(rs), rs}
	paths, e := exportReport(dir, doc, html)
	if e != nil {
		return usageError(out, "report export failed: "+e.Error())
	}
	if a.JSON {
		_ = json.NewEncoder(out).Encode(map[string]any{"status": doc.Status, "files": paths})
	} else {
		fmt.Fprintf(out, "[%s] Reports: %s\n", doc.Status, strings.Join(paths, ", "))
	}
	return exitCode(doc.Status)
}
func exportReport(dir string, doc Report, html bool) ([]string, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	suffix := make([]byte, 6)
	if _, e := rand.Read(suffix); e != nil {
		return nil, e
	}
	stem := "aslunix-" + doc.Time.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(suffix)
	jsonBytes, e := json.MarshalIndent(doc, "", "  ")
	if e != nil {
		return nil, e
	}
	var text bytes.Buffer
	fmt.Fprintf(&text, "AslUnix %s | %s | %s | %s\n\n", doc.Version, safeText(doc.Host), doc.Time.Format(time.RFC3339), doc.Status)
	for _, r := range doc.Results {
		render(&text, r, false)
		fmt.Fprintln(&text, "Raw diagnostic:", safeText(r.Raw))
		fmt.Fprintln(&text)
	}
	payloads := map[string][]byte{".json": jsonBytes, ".txt": text.Bytes()}
	extensions := []string{".json", ".txt"}
	if html {
		tmpl := template.Must(template.New("report").Funcs(template.FuncMap{"pretty": func(v any) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }}).Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>AslUnix report</title><style>body{font:15px system-ui;max-width:1100px;margin:2rem auto;padding:1rem;background:#111827;color:#e5e7eb}section{border:1px solid #4b5563;padding:1rem;margin:1rem 0}pre{white-space:pre-wrap;overflow-wrap:anywhere}h1,h2{color:#93c5fd}</style><h1>AslUnix {{.Version}}</h1><p>{{.Host}} | {{.Time}} | {{.Status}}</p>{{range .Results}}<section><h2>{{.Check}} — {{.Status}}</h2><p>{{.Summary}}</p><p>{{.Time}} | {{.Source}}</p><pre>{{pretty .Data}}</pre>{{range .Limitations}}<p>Limitation: {{.}}</p>{{end}}<details><summary>Raw diagnostic</summary><pre>{{.Raw}}</pre></details></section>{{end}}</html>`))
		var b bytes.Buffer
		if e = tmpl.Execute(&b, doc); e != nil {
			return nil, e
		}
		payloads[".html"] = b.Bytes()
		extensions = append(extensions, ".html")
	}
	paths := []string{}
	cleanup := func() {
		for _, p := range paths {
			os.Remove(p)
		}
	}
	for _, ext := range extensions {
		path, e := filepath.Abs(filepath.Join(dir, stem+ext))
		if e != nil {
			cleanup()
			return nil, e
		}
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			cleanup()
			return nil, e
		}
		paths = append(paths, path)
		_, e = f.Write(payloads[ext])
		ce := f.Close()
		if e != nil || ce != nil {
			cleanup()
			if e != nil {
				return nil, e
			}
			return nil, ce
		}
	}
	return paths, nil
}

var reportName = regexp.MustCompile(`^aslunix-[0-9]{8}T[0-9]{6}Z-[a-f0-9]{12}\.(json|txt|html)$`)

func retainedCandidates(dir string, days int, now time.Time) ([]string, error) {
	if days == 0 {
		return nil, fmt.Errorf("retention is disabled (report_retention_days=0)")
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return nil, e
	}
	paths := []string{}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	for _, d := range entries {
		if !reportName.MatchString(d.Name()) || d.Type()&os.ModeSymlink != 0 || d.IsDir() {
			continue
		}
		info, e := d.Info()
		if e != nil {
			return nil, e
		}
		if info.ModTime().Before(cutoff) {
			paths = append(paths, filepath.Join(dir, d.Name()))
		}
	}
	return paths, nil
}
func (a *App) pruneReports(dir string, dry, yes bool, input *bufio.Reader, out io.Writer) int {
	paths, e := retainedCandidates(dir, a.Config.ReportRetentionDays, time.Now())
	if e != nil {
		return usageError(out, e.Error())
	}
	for _, p := range paths {
		fmt.Fprintln(out, "Delete:", safeText(p))
	}
	if dry || len(paths) == 0 {
		return 0
	}
	if !confirm(input, out, yes, fmt.Sprintf("delete %d expired AslUnix reports", len(paths))) {
		fmt.Fprintln(out, "Cancelled")
		return 0
	}
	for _, p := range paths {
		if e = os.Remove(p); e != nil {
			return usageError(out, e.Error())
		}
	}
	return 0
}
