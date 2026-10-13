package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

type fakeRunner struct {
	data   string
	err    error
	limits []string
}

func (f fakeRunner) Collect(ctx context.Context, name string, input map[string]any) (json.RawMessage, string, []string, error) {
	return json.RawMessage(f.data), "raw-fixture", f.limits, f.err
}
func testApp(f fakeRunner) *App { return &App{Config: defaultConfig(), Runner: f} }
func TestDiskThresholds(t *testing.T) {
	for _, tc := range []struct {
		free, size float64
		want       Status
	}{{30, 100, OK}, {20, 100, Warning}, {10, 100, Critical}, {9, 100, Critical}, {7, 20, Warning}, {4, 20, Critical}, {0, 0, Unknown}, {200, 100, Unknown}} {
		t.Run(fmt.Sprint(tc.free, "/", tc.size), func(t *testing.T) {
			a := testApp(fakeRunner{data: fmt.Sprintf(`[{"Volume":"C:","SizeBytes":%f,"FreeBytes":%f}]`, tc.size*(1<<30), tc.free*(1<<30))})
			r := a.collect(context.Background(), "disks", nil)
			if r.Status != tc.want {
				t.Fatalf("%s, want %s: %s", r.Status, tc.want, r.Summary)
			}
			if r.Raw != "raw-fixture" || r.Time.IsZero() || r.Source == "" {
				t.Fatal("diagnostic metadata missing")
			}
		})
	}
}
func TestCollectorFailuresNeverHealthy(t *testing.T) {
	for _, f := range []fakeRunner{{err: errors.New("access denied")}, {err: context.DeadlineExceeded}, {data: `[]`}, {data: `bad json`}, {data: `[{"CPUPercent":null,"MemoryPercent":12}]`}, {data: `[{}]`}, {data: `[{"CPUPercent":5,"MemoryPercent":20}]`, limits: []string{"INCOMPLETE: counter unavailable"}}} {
		r := testApp(f).collect(context.Background(), "resources", nil)
		if r.Status != Unknown {
			t.Fatalf("%+v yielded %s", f, r.Status)
		}
	}
}
func TestResourcePolicy(t *testing.T) {
	for _, tc := range []struct {
		cpu, mem int
		want     Status
	}{{20, 20, OK}, {85, 20, Warning}, {20, 95, Critical}, {-1, 20, Unknown}} {
		r := testApp(fakeRunner{data: fmt.Sprintf(`[{"CPUPercent":%d,"MemoryPercent":%d}]`, tc.cpu, tc.mem)}).collect(context.Background(), "resources", nil)
		if r.Status != tc.want {
			t.Fatalf("%+v: %s", tc, r.Status)
		}
	}
}
func TestRequiredServices(t *testing.T) {
	a := testApp(fakeRunner{data: `[{"Name":"DCSService","State":"Stopped"}]`})
	a.Config.RequiredServices = []string{"DCSService"}
	if r := a.collect(context.Background(), "services", nil); r.Status != Critical {
		t.Fatal(r)
	}
	a.Runner = fakeRunner{data: `[{"Name":"dcsservice","State":"Running"}]`}
	if r := a.collect(context.Background(), "services", nil); r.Status != OK {
		t.Fatal(r)
	}
	a.Runner = fakeRunner{data: `[]`}
	if r := a.collect(context.Background(), "services", nil); r.Status != Critical {
		t.Fatal(r)
	}
}
func TestAdminBaselineOnlyDirect(t *testing.T) {
	a := testApp(fakeRunner{data: `[{"SID":"S-1-5-21-1","Kind":"Group","Membership":"direct"},{"SID":"S-1-5-21-2","Kind":"user","Membership":"resolved_domain"}]`})
	a.Config.ApprovedAdminSIDs = []string{"S-1-5-21-1"}
	if r := a.collect(context.Background(), "admins", nil); r.Status != OK {
		t.Fatal(r)
	}
	a.Config.ApprovedAdminSIDs = []string{"S-1-5-21-3"}
	if r := a.collect(context.Background(), "admins", nil); r.Status != Warning {
		t.Fatal(r)
	}
}
func TestRAIDCannotInferHealth(t *testing.T) {
	for _, tc := range []struct {
		json string
		want Status
	}{{`{}`, Unknown}, {`{"Controller Status":"Optimal","VD LIST":[{"State":"Optl"}],"PD LIST":[{"State":"Onln"}]}`, OK}, {`{"Controller Status":"Optimal","VD LIST":[{"State":"Dgrd"}],"PD LIST":[{"State":"Rbld"}]}`, Critical}, {`{"Controller Status":"Optimal","VD LIST":[{"State":"Optl"}],"PD LIST":[{"State":"new firmware word"}]}`, Unknown}, {`{"Controller Status":"Optimal"}`, Unknown}} {
		var v any
		_ = json.Unmarshal([]byte(tc.json), &v)
		s, _ := classifyRAID(v)
		if s != tc.want {
			t.Fatalf("%s => %s", tc.json, s)
		}
	}
}
func TestAggregateAndExitCodes(t *testing.T) {
	if worst([]Result{{Status: OK}, {Status: Unknown}}) != Unknown {
		t.Fatal("unknown hidden")
	}
	for s, code := range map[Status]int{OK: 0, NotApplicable: 0, Warning: 1, Critical: 2, Unknown: 3} {
		if exitCode(s) != code {
			t.Fatal(s)
		}
	}
}
