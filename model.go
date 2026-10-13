package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

type Status string

const (
	OK            Status = "OK"
	Warning       Status = "WARNING"
	Critical      Status = "CRITICAL"
	Unknown       Status = "UNKNOWN"
	NotApplicable Status = "NOT_APPLICABLE"
)

type Result struct {
	Check       string    `json:"check"`
	Status      Status    `json:"status"`
	Time        time.Time `json:"checked_at"`
	Source      string    `json:"source"`
	Summary     string    `json:"summary"`
	Data        any       `json:"data,omitempty"`
	Raw         string    `json:"raw,omitempty"`
	Limitations []string  `json:"limitations,omitempty"`
}

func result(name string, s Status, source, msg string, data any) Result {
	return Result{Check: name, Status: s, Time: time.Now().UTC(), Source: source, Summary: msg, Data: data}
}
func worst(results []Result) Status {
	rank := map[Status]int{NotApplicable: 0, OK: 1, Unknown: 2, Warning: 3, Critical: 4}
	s := NotApplicable
	for _, r := range results {
		if rank[r.Status] > rank[s] {
			s = r.Status
		}
	}
	return s
}
func exitCode(s Status) int {
	switch s {
	case OK, NotApplicable:
		return 0
	case Warning:
		return 1
	case Critical:
		return 2
	default:
		return 3
	}
}

type Config struct {
	DiskWarningPercent    float64           `json:"disk_warning_free_percent"`
	DiskCriticalPercent   float64           `json:"disk_critical_free_percent"`
	DiskWarningGB         float64           `json:"disk_warning_free_gb"`
	DiskCriticalGB        float64           `json:"disk_critical_free_gb"`
	CPUWarningPercent     float64           `json:"cpu_warning_percent"`
	CPUCriticalPercent    float64           `json:"cpu_critical_percent"`
	MemoryWarningPercent  float64           `json:"memory_warning_percent"`
	MemoryCriticalPercent float64           `json:"memory_critical_percent"`
	RequiredServices      []string          `json:"required_services"`
	ApprovedAdminSIDs     []string          `json:"approved_admin_sids"`
	TimeoutSeconds        int               `json:"timeout_seconds"`
	SampleSeconds         int               `json:"sample_seconds"`
	WatchSeconds          int               `json:"watch_seconds"`
	ReportRetentionDays   int               `json:"report_retention_days"`
	ReportDirectory       string            `json:"report_directory"`
	Tools                 map[string]string `json:"tools"`
}

func defaultConfig() Config {
	return Config{DiskWarningPercent: 20, DiskCriticalPercent: 10, DiskWarningGB: 10, DiskCriticalGB: 5, CPUWarningPercent: 85, CPUCriticalPercent: 95, MemoryWarningPercent: 85, MemoryCriticalPercent: 95, TimeoutSeconds: 30, SampleSeconds: 1, WatchSeconds: 30, ReportRetentionDays: 30, ReportDirectory: "reports", Tools: map[string]string{}}
}
func loadConfig(path string) (Config, error) {
	c := defaultConfig()
	if path != "" {
		b, e := os.ReadFile(path)
		if e != nil {
			return c, e
		}
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&c); e != nil {
			return c, e
		}
		var extra any
		if e = decoder.Decode(&extra); e != io.EOF {
			return c, fmt.Errorf("configuration must contain exactly one JSON object")
		}
	}
	if c.DiskCriticalPercent < 0 || c.DiskWarningPercent > 100 || c.DiskCriticalPercent >= c.DiskWarningPercent || c.DiskCriticalGB < 0 || c.DiskWarningGB < c.DiskCriticalGB {
		return c, fmt.Errorf("invalid disk thresholds: critical must be below warning")
	}
	if c.CPUWarningPercent <= 0 || c.CPUCriticalPercent > 100 || c.CPUCriticalPercent <= c.CPUWarningPercent || c.MemoryWarningPercent <= 0 || c.MemoryCriticalPercent > 100 || c.MemoryCriticalPercent <= c.MemoryWarningPercent {
		return c, fmt.Errorf("invalid resource thresholds")
	}
	if c.TimeoutSeconds < 1 || c.TimeoutSeconds > 3600 || c.SampleSeconds < 1 || c.SampleSeconds >= c.TimeoutSeconds || c.WatchSeconds < 1 || c.ReportRetentionDays < 0 || c.ReportRetentionDays > 36500 {
		return c, fmt.Errorf("invalid timing or retention configuration")
	}
	for _, name := range c.RequiredServices {
		if name == "" {
			return c, fmt.Errorf("required_services contains an empty name")
		}
	}
	for _, sid := range c.ApprovedAdminSIDs {
		if !regexp.MustCompile(`^S-1-[0-9]+(-[0-9]+)+$`).MatchString(sid) {
			return c, fmt.Errorf("invalid approved administrator SID %q", sid)
		}
	}
	return c, nil
}
