package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "0.1.0"

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout)) }
func run(args []string, in io.Reader, out io.Writer) int {
	configPath := ""
	jsonMode := false
	color := false
	for len(args) > 0 {
		switch args[0] {
		case "--config":
			if len(args) < 2 {
				fmt.Fprintln(out, "--config requires a path")
				return 4
			}
			configPath = args[1]
			args = args[2:]
		case "--json":
			jsonMode = true
			args = args[1:]
		case "--color":
			color = true
			args = args[1:]
		case "--no-color":
			color = false
			args = args[1:]
		case "--version":
			fmt.Fprintln(out, "AslUnix", version)
			return 0
		default:
			goto parsed
		}
	}
parsed:
	c, e := loadConfig(configPath)
	if e != nil {
		fmt.Fprintln(out, "Configuration error:", e)
		return 4
	}
	color, restoreColor := colorMode(out, color)
	defer restoreColor()
	a := &App{Config: c, Runner: PowerShellRunner{}, JSON: jsonMode, Color: color}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	input := bufio.NewReader(in)
	if len(args) == 0 {
		brandStandaloneConsole()
		return a.shell(ctx, input, out)
	}
	return a.execute(ctx, args, input, out)
}
func (a *App) execute(ctx context.Context, args []string, input *bufio.Reader, out io.Writer) int {
	if len(args) == 0 {
		return 0
	}
	name := strings.ToLower(args[0])
	rest := args[1:]
	if name == "help" {
		topic := ""
		if len(rest) > 0 {
			topic = rest[0]
		}
		writeHelp(out, topic)
		return 0
	}
	if name == "version" {
		fmt.Fprintln(out, "AslUnix", version)
		return 0
	}
	if name == "health" {
		if len(rest) > 0 {
			return usageError(out, "health takes no arguments")
		}
		return a.show(out, a.health(ctx))
	}
	if name == "report" {
		return a.report(ctx, rest, input, out)
	}
	if name == "watch" {
		return a.watch(ctx, rest, input, out)
	}
	if name == "follow" {
		if a.JSON {
			return usageError(out, "follow streams text and cannot be used with --json")
		}
		return follow(ctx, rest, out)
	}
	if isFileCommand(name) {
		r := a.fileCommand(ctx, name, rest, input, out)
		return a.show(out, []Result{r})
	}
	if isProbe(name) {
		r := a.probe(ctx, name, rest)
		return a.show(out, []Result{r})
	}
	if name == "raid" || name == "tools" || name == "capture" {
		return a.show(out, []Result{a.integration(ctx, name, rest)})
	}
	if name == "df" {
		name = "disks"
	}
	if collectorNames[name] {
		p, e := collectorOptions(name, rest)
		if e != nil {
			return usageError(out, e.Error())
		}
		return a.show(out, []Result{a.collect(ctx, name, p)})
	}
	return usageError(out, "Unknown command: "+name+". Use help.")
}
func usageError(out io.Writer, s string) int { fmt.Fprintln(out, s); return 4 }
func collectorOptions(name string, args []string) (map[string]any, error) {
	p := map[string]any{}
	allowed := map[string]string{}
	switch name {
	case "admins":
		allowed["expand-domain"] = "bool"
	case "events":
		allowed = map[string]string{"hours": "value", "since": "value", "until": "value", "id": "value", "log": "value", "limit": "value"}
		p["hours"] = 24
		p["limit"] = 100
	case "wsus":
		allowed["test"] = "bool"
	case "vmware", "sql":
		allowed["server"] = "value"
		if name == "sql" {
			allowed["backup"] = "value"
		}
	}
	positional := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			kind, ok := allowed[key]
			if !ok {
				return nil, fmt.Errorf("unsupported option %s for %s", arg, name)
			}
			if kind == "bool" {
				p[strings.ReplaceAll(key, "-", "_")] = true
			} else {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("%s requires a value", arg)
				}
				i++
				p[key] = args[i]
			}
		} else {
			positional = append(positional, arg)
		}
	}
	if name == "permissions" || name == "signature" {
		if len(positional) != 1 {
			return nil, fmt.Errorf("%s requires one quoted path", name)
		}
		abs, e := filepath.Abs(positional[0])
		if e != nil {
			return nil, e
		}
		p["path"] = abs
	} else if name == "registry" {
		if len(positional) > 1 {
			return nil, fmt.Errorf("registry accepts at most one path")
		}
		if len(positional) == 1 {
			p["path"] = positional[0]
		}
	} else if len(positional) > 0 {
		return nil, fmt.Errorf("unexpected arguments for %s", name)
	}
	if name == "events" {
		for _, k := range []string{"hours", "limit"} {
			if s, ok := p[k].(string); ok {
				n, e := strconv.Atoi(s)
				if e != nil || n < 1 || n > 10000 {
					return nil, fmt.Errorf("%s must be between 1 and 10000", k)
				}
				p[k] = n
			}
		}
		for _, k := range []string{"since", "until"} {
			if s, ok := p[k].(string); ok {
				if _, e := time.Parse(time.RFC3339, s); e != nil {
					return nil, fmt.Errorf("%s requires RFC3339 time with timezone", k)
				}
			}
		}
		if s, ok := p["id"].(string); ok {
			for _, v := range strings.Split(s, ",") {
				n, e := strconv.Atoi(v)
				if e != nil || n < 0 || n > 65535 {
					return nil, fmt.Errorf("invalid event ID %q", v)
				}
			}
		}
	}
	return p, nil
}
func (a *App) show(out io.Writer, rs []Result) int {
	if a.JSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if e := enc.Encode(rs); e != nil {
			return 3
		}
	} else {
		for _, r := range rs {
			render(out, r, a.Color)
		}
	}
	return exitCode(worst(rs))
}
func (a *App) watch(ctx context.Context, args []string, input *bufio.Reader, out io.Writer) int {
	interval := a.Config.WatchSeconds
	count := 0
	names := []string{"health"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		names = strings.Split(args[0], ",")
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		if (args[i] != "--interval" && args[i] != "--count") || i+1 >= len(args) {
			return usageError(out, "watch [health|disks,resources,...] [--interval SECONDS] [--count N]")
		}
		n, e := strconv.Atoi(args[i+1])
		if e != nil || n < 1 || n > 86400 {
			return usageError(out, "watch values must be integers from 1 to 86400")
		}
		if args[i] == "--interval" {
			interval = n
		} else {
			count = n
		}
		i++
	}
	for _, n := range names {
		if n != "health" && n != "raid" && !collectorNames[n] {
			return usageError(out, "watch only accepts read-only collector names")
		}
		if n == "permissions" || n == "signature" || n == "vmware" || n == "sql" {
			return usageError(out, "watch does not accept collectors requiring arguments")
		}
	}
	last := 0
	for i := 0; count == 0 || i < count; i++ {
		if ctx.Err() != nil {
			break
		}
		rs := []Result{}
		for _, n := range names {
			if n == "health" {
				rs = append(rs, a.health(ctx)...)
			} else if n == "raid" {
				rs = append(rs, a.integration(ctx, n, nil))
			} else {
				p, _ := collectorOptions(n, nil)
				rs = append(rs, a.collect(ctx, n, p))
			}
		}
		last = a.show(out, rs)
		if count > 0 && i+1 == count {
			break
		}
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return last
		case <-timer.C:
		}
	}
	return last
}
