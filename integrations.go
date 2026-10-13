package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var integrationCandidates = map[string][]string{"perccli": {"perccli64.exe", "perccli.exe"}, "tshark": {"tshark.exe"}, "sysinternals": {"sigcheck64.exe", "sigcheck.exe"}, "sqlcmd": {"sqlcmd.exe"}}

func (a *App) tool(name string) (string, error) {
	if p := a.Config.Tools[name]; p != "" {
		info, e := os.Stat(p)
		if e != nil || info.IsDir() {
			return "", fmt.Errorf("configured %s tool unavailable at %s", name, p)
		}
		abs, e := filepath.Abs(p)
		return abs, e
	}
	for _, candidate := range integrationCandidates[name] {
		if p, e := exec.LookPath(candidate); e == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not installed or configured; obtain from the vendor through your approved process, then set tools.%s to its full executable path", name, name)
}
func (a *App) integration(parent context.Context, name string, args []string) Result {
	r := result(name, Unknown, "Installed vendor tool", "Integration unavailable", nil)
	ctx, cancel := context.WithTimeout(parent, time.Duration(a.Config.TimeoutSeconds)*time.Second)
	defer cancel()
	if name == "tools" {
		if len(args) != 0 {
			r.Summary = "tools takes no arguments"
			return r
		}
		rows := []map[string]any{}
		r.Status = OK
		r.Summary = "Tool discovery only"
		for _, tool := range []string{"perccli", "tshark", "sysinternals", "sqlcmd"} {
			path, e := a.tool(tool)
			status := OK
			detail := "found"
			if e != nil {
				status = Unknown
				detail = e.Error()
				r.Status = Unknown
			}
			rows = append(rows, map[string]any{"Tool": tool, "Path": path, "Status": status, "Details": detail})
		}
		r.Data = rows
		r.Limitations = []string{"PowerCLI and SqlServer modules are detected by their commands on Windows; discovery does not establish tool compatibility or redistribution rights"}
		return r
	}
	tool := "perccli"
	if name == "capture" {
		tool = "tshark"
	}
	path, e := a.tool(tool)
	if e != nil {
		r.Summary = e.Error()
		return r
	}
	r.Source = path
	toolArgs := []string{"/call", "show", "all", "J"}
	if name == "raid" && len(args) > 0 {
		r.Summary = "raid accepts no arguments; only read-only PERCCLI show commands are supported"
		return r
	}
	if name == "capture" {
		if len(args) == 0 {
			toolArgs = []string{"-D"}
			r.Summary = "Capture interfaces"
		} else {
			p := map[string]string{}
			for i := 0; i < len(args); i += 2 {
				if i+1 >= len(args) || !contains([]string{"--interface", "--seconds", "--out"}, args[i]) {
					r.Summary = "capture [--interface ID --seconds N --out FILE.pcapng]"
					return r
				}
				p[args[i]] = args[i+1]
			}
			secs, e := strconv.Atoi(p["--seconds"])
			iface, e2 := strconv.Atoi(p["--interface"])
			if e != nil || e2 != nil || secs < 1 || secs > 300 || iface < 1 || p["--out"] == "" {
				r.Summary = "capture needs a numeric interface, 1..300 seconds and an output path"
				return r
			}
			output, e := filepath.Abs(p["--out"])
			if e != nil {
				r.Summary = e.Error()
				return r
			}
			f, e := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if e != nil {
				r.Summary = e.Error()
				return r
			}
			f.Close()
			// Reserving the output prevents accidentally overwriting an existing capture.
			toolArgs = []string{"-i", strconv.Itoa(iface), "-a", fmt.Sprintf("duration:%d", secs), "-w", output}
			r.Summary = "Bounded capture requested; may require Npcap/admin rights"
		}
	}
	b, stderr, e := runBounded(ctx, path, toolArgs, os.Environ())
	r.Raw = string(b) + "\n" + stderr
	if e != nil {
		r.Summary = "Vendor tool failed: " + e.Error()
		return r
	}
	if name == "capture" {
		r.Status = OK
		r.Data = string(b)
		r.Limitations = []string{"Packet capture is explicit and writes the requested file; effective duration is capped by timeout_seconds too"}
		return r
	}
	var data any
	if e = json.Unmarshal(b, &data); e != nil {
		r.Summary = "PERCCLI returned unsupported/non-JSON output"
		return r
	}
	r.Data = data
	st, known := classifyRAID(data)
	r.Status = st
	r.Summary = fmt.Sprintf("PERCCLI status fields interpreted: %d", known)
	r.Limitations = []string{"Controller/virtual/physical state vocabulary varies by PERCCLI and firmware; unrecognized or absent fields are UNKNOWN; validate on actual Dell hardware"}
	return r
}
func classifyRAID(data any) (Status, int) {
	if root, ok := data.(map[string]any); ok {
		if children, exists := root["Controllers"]; exists {
			list, ok := children.([]any)
			if !ok || len(list) == 0 {
				return Unknown, 0
			}
			rs := []Result{}
			count := 0
			for _, child := range list {
				status, n := classifyRAID(child)
				rs = append(rs, Result{Status: status})
				count += n
			}
			return worst(rs), count
		}
	}
	s := OK
	known := 0
	controller, virtual, physical := false, false, false
	var visit func(any, string)
	visit = func(v any, parent string) {
		switch x := v.(type) {
		case []any:
			for _, c := range x {
				visit(c, parent)
			}
		case map[string]any:
			for k, value := range x {
				if k == "VD LIST" {
					if list, ok := value.([]any); ok && len(list) > 0 {
						virtual = true
					}
				}
				if k == "PD LIST" {
					if list, ok := value.([]any); ok && len(list) > 0 {
						physical = true
					}
				}
				if k == "Controller Status" {
					controller = true
				}
				if k == "State" || k == "Controller Status" {
					text, ok := value.(string)
					if !ok {
						s = worst([]Result{{Status: s}, {Status: Unknown}})
						continue
					}
					state := Unknown
					switch strings.ToLower(strings.TrimSpace(text)) {
					case "optimal", "optl", "onln", "online":
						state = OK
					case "rbld", "rebuild", "rebuilding", "ugood", "ghs", "dhs":
						state = Warning
					case "dgrd", "degraded", "offln", "offline", "failed", "f", "ubad", "pdgd":
						state = Critical
					}
					known++
					s = worst([]Result{{Status: s}, {Status: state}})
				}
				// Command status failure is a collector failure even if stale health fields look good.
				if parent == "Command Status" && k == "Status" {
					if text, ok := value.(string); !ok || !strings.EqualFold(text, "Success") {
						s = worst([]Result{{Status: s}, {Status: Unknown}})
					}
				}
				visit(value, k)
			}
		}
	}
	visit(data, "")
	if known == 0 || !controller || !virtual || !physical {
		s = worst([]Result{{Status: s}, {Status: Unknown}})
	}
	return s, known
}
