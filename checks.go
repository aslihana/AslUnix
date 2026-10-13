package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type App struct {
	Config Config
	Runner Runner
	Color  bool
	JSON   bool
}

var healthChecks = []string{"disks", "resources", "services", "admins", "raid"}
var collectorNames = map[string]bool{"disks": true, "resources": true, "services": true, "accounts": true, "admins": true, "permissions": true, "identity": true, "events": true, "network": true, "proxy": true, "time": true, "domain": true, "os": true, "updates": true, "drivers": true, "tasks": true, "hardware": true, "registry": true, "wsus": true, "iis": true, "dotnet": true, "signature": true, "vmware": true, "sql": true}

func (a *App) collect(ctx context.Context, name string, input map[string]any) Result {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(a.Config.TimeoutSeconds)*time.Second)
	defer cancel()
	if input == nil {
		input = map[string]any{}
	}
	input["sample_seconds"] = a.Config.SampleSeconds
	d, raw, limits, e := a.Runner.Collect(ctx, name, input)
	r := result(name, OK, "Windows PowerShell / CIM", "Inspection completed", d)
	r.Raw = raw
	r.Limitations = limits
	if e != nil {
		r.Status = Unknown
		r.Summary = e.Error()
		return r
	}
	incomplete := false
	for _, limit := range limits {
		if strings.HasPrefix(limit, "INCOMPLETE:") {
			incomplete = true
		}
	}
	if incomplete {
		r.Status = Unknown
		r.Summary = "Inspection incomplete; see limitations"
	}
	switch name {
	case "disks":
		var rows []struct {
			Volume    string
			SizeBytes float64
			FreeBytes float64
		}
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		if len(rows) == 0 {
			r.Status = Unknown
			r.Summary = "No fixed volumes returned"
			break
		}
		table := []map[string]any{}
		for _, v := range rows {
			st := OK
			pct := 0.0
			gb := v.FreeBytes / (1 << 30)
			if v.SizeBytes <= 0 || v.FreeBytes < 0 || v.FreeBytes > v.SizeBytes {
				st = Unknown
			} else {
				pct = v.FreeBytes / v.SizeBytes * 100
				if pct <= a.Config.DiskCriticalPercent || gb <= a.Config.DiskCriticalGB {
					st = Critical
				} else if pct <= a.Config.DiskWarningPercent || gb <= a.Config.DiskWarningGB {
					st = Warning
				}
			}
			r.Status = worst([]Result{r, {Status: st}})
			table = append(table, map[string]any{"Volume": v.Volume, "SizeGiB": v.SizeBytes / (1 << 30), "FreeGiB": gb, "FreePercent": pct, "Status": st})
		}
		r.Data = table
		r.Summary = "Fixed volume capacity; GB thresholds use binary GiB"
	case "resources":
		var rows []struct {
			CPUPercent    *float64
			MemoryPercent *float64
		}
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		if len(rows) == 0 {
			r.Status = Unknown
			r.Summary = "No resource sample returned"
			break
		}
		for _, v := range rows {
			s := OK
			if v.CPUPercent == nil || v.MemoryPercent == nil {
				s = Unknown
			} else if *v.CPUPercent < 0 || *v.CPUPercent > 100 || *v.MemoryPercent < 0 || *v.MemoryPercent > 100 {
				s = Unknown
			} else if *v.CPUPercent >= a.Config.CPUCriticalPercent || *v.MemoryPercent >= a.Config.MemoryCriticalPercent {
				s = Critical
			} else if *v.CPUPercent >= a.Config.CPUWarningPercent || *v.MemoryPercent >= a.Config.MemoryWarningPercent {
				s = Warning
			}
			r.Status = worst([]Result{r, {Status: s}})
		}
		r.Summary = "Sampled CPU and physical memory; process CPU is normalized across logical CPUs"
	case "services":
		var rows []struct {
			Name  string
			State string
		}
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		missing := []string{}
		for _, required := range a.Config.RequiredServices {
			found := false
			for _, s := range rows {
				if strings.EqualFold(s.Name, required) && strings.EqualFold(s.State, "Running") {
					found = true
				}
			}
			if !found {
				missing = append(missing, required)
			}
		}
		if len(missing) > 0 {
			r.Status = Critical
			r.Summary = "Required services missing or stopped: " + strings.Join(missing, ", ")
		} else if len(a.Config.RequiredServices) == 0 {
			r.Summary = "Service inventory; no required-services policy configured"
			r.Limitations = append(r.Limitations, "Inventory is not an application health assertion")
		}
	case "admins":
		var rows []struct {
			SID        string
			Kind       string
			Membership string
		}
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		if len(rows) == 0 {
			r.Status = Unknown
			r.Summary = "No direct administrator membership returned"
			break
		}
		extra := []string{}
		for _, v := range rows {
			if v.SID == "" || v.Kind == "Unresolved" {
				r.Status = worst([]Result{r, {Status: Unknown}})
			}
			if v.Membership != "direct" {
				continue
			}
			if v.Kind == "Group" {
				r.Limitations = append(r.Limitations, "Direct group "+v.SID+" may contain additional privileged identities; use --expand-domain for AD group resolution")
			}
			if len(a.Config.ApprovedAdminSIDs) > 0 && !contains(a.Config.ApprovedAdminSIDs, v.SID) {
				extra = append(extra, v.SID)
			}
		}
		r.Summary = "Direct membership of S-1-5-32-544; not every privileged identity"
		if len(extra) > 0 {
			r.Status = Warning
			r.Summary = "Administrator SIDs outside baseline: " + strings.Join(extra, ", ")
		}
	case "signature":
		var rows []struct{ Status string }
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		if len(rows) == 0 {
			r.Status = Unknown
		}
		for _, row := range rows {
			st := Unknown
			switch row.Status {
			case "Valid":
				st = OK
			case "NotSigned":
				st = Warning
			case "HashMismatch", "NotTrusted":
				st = Critical
			}
			r.Status = worst([]Result{r, {Status: st}})
		}
		r.Summary = "Authenticode signature inspection; offline revocation may be incomplete"
	case "domain":
		var rows []struct{ PartOfDomain bool }
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		if len(rows) == 1 && !rows[0].PartOfDomain {
			r.Status = NotApplicable
			r.Summary = "Workgroup machine; domain connectivity not applicable"
		}
	case "wsus":
		var rows []json.RawMessage
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		if len(rows) == 0 && !incomplete {
			r.Status = NotApplicable
			r.Summary = "No WSUS policy configured"
		}

	case "events":
		var rows []struct{ Level int }
		if e = json.Unmarshal(d, &rows); e != nil {
			return invalidData(r, e)
		}
		for _, v := range rows {
			if v.Level <= 2 {
				r.Status = worst([]Result{r, {Status: Critical}})
			} else if v.Level == 3 {
				r.Status = worst([]Result{r, {Status: Warning}})
			}
		}
		r.Summary = fmt.Sprintf("%d matching events; severity reflects logged events, not current fault state", len(rows))
	}
	return r
}
func invalidData(r Result, e error) Result {
	r.Status = Unknown
	r.Summary = "Invalid collector data: " + e.Error()
	return r
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}
func (a *App) health(ctx context.Context) []Result {
	r := []Result{}
	for _, name := range healthChecks {
		if ctx.Err() != nil {
			r = append(r, result(name, Unknown, "context", ctx.Err().Error(), nil))
			continue
		}
		if name == "raid" {
			r = append(r, a.integration(ctx, name, nil))
		} else {
			r = append(r, a.collect(ctx, name, nil))
		}
	}
	return r
}
