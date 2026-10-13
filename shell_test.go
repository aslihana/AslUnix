package main

import (
	"reflect"
	"testing"
)

func TestQuotedWindowsPaths(t *testing.T) {
	for line, want := range map[string][]string{`permissions "C:\Program Files\DCS"`: {"permissions", `C:\Program Files\DCS`}, `cd 'C:\logs\'`: {"cd", `C:\logs\`}, `grep "can't" "C:\a b.txt"`: {"grep", "can't", `C:\a b.txt`}, `cat ""`: {"cat", ""}, `grep "say ""yes""" f`: {"grep", `say "yes"`, "f"}, `cat $(whoami)`: {"cat", "$(whoami)"}} {
		got, e := tokenize(line)
		if e != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%q => %#v, %v", line, got, e)
		}
	}
	if _, e := tokenize(`cd "C:\x`); e == nil {
		t.Fatal("unclosed quote accepted")
	}
}
func TestCollectorOptionsRejectInjectionAndTypos(t *testing.T) {
	for _, args := range [][]string{{"--id", "1;Remove-Item C:"}, {"--hours", "-1"}, {"--limit", "0"}, {"--since", "yesterday"}, {"--unknown"}} {
		if _, e := collectorOptions("events", args); e == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	p, e := collectorOptions("permissions", []string{`C:\a';Write-Host oops;#`})
	if e != nil || p["path"] == nil {
		t.Fatal("path should remain data", e)
	}
}
func TestCompletion(t *testing.T) {
	if got := completions("hea"); !reflect.DeepEqual(got, []string{"head", "health"}) {
		t.Fatal(got)
	}
}
