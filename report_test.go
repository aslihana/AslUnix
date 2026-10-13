package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReportExportsEscapedHTMLAndRawJSON(t *testing.T) {
	dir := t.TempDir()
	doc := Report{Version: version, Host: "fixture", Time: time.Now().UTC(), Status: Unknown, Results: []Result{result("test", Unknown, "fixture", "<script>alert(1)</script>", map[string]any{"x": "<img src=x onerror=bad>"})}}
	doc.Results[0].Raw = "raw-details"
	paths, e := exportReport(dir, doc, true)
	if e != nil || len(paths) != 3 {
		t.Fatal(paths, e)
	}
	for _, p := range paths {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		if strings.HasSuffix(p, ".html") && (strings.Contains(string(b), "<script>") || strings.Contains(string(b), "<img src=x")) {
			t.Fatal("unescaped HTML")
		}
		if strings.HasSuffix(p, ".json") {
			var decoded Report
			if e = json.Unmarshal(b, &decoded); e != nil || decoded.Results[0].Raw != "raw-details" || decoded.Status != Unknown {
				t.Fatal(e)
			}
		}
	}
}
func TestRetentionOnlyOwnExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-40 * 24 * time.Hour)
	owned := filepath.Join(dir, "aslunix-20200101T000000Z-abcdef012345.json")
	other := filepath.Join(dir, "important.json")
	recent := filepath.Join(dir, "aslunix-20260101T000000Z-abcdef012345.txt")
	for _, p := range []string{owned, other, recent} {
		os.WriteFile(p, []byte("keep"), 0600)
	}
	os.Chtimes(owned, old, old)
	os.Chtimes(other, old, old)
	paths, e := retainedCandidates(dir, 30, time.Now())
	if e != nil || len(paths) != 1 || paths[0] != owned {
		t.Fatal(paths, e)
	}
}
