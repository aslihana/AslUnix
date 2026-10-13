package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigRejectsDangerousThresholdsAndTypos(t *testing.T) {
	for _, text := range []string{`{"disk_critical_free_percent":30}`, `{"disk_warning_free_percent":101}`, `{"timeout_seconds":0}`, `{"sample_seconds":30}`, `{"cpu_critical_percent":80}`, `{"disk_warning_fre_percent":5}`, `{"approved_admin_sids":["domain\\user"]}`} {
		p := filepath.Join(t.TempDir(), "config.json")
		os.WriteFile(p, []byte(text), 0600)
		if _, e := loadConfig(p); e == nil {
			t.Fatalf("accepted %s", text)
		}
	}
	if _, e := loadConfig("aslunix.example.json"); e != nil {
		t.Fatal(e)
	}
}
