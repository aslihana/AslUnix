package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// Tokenize preserves Windows backslashes. Quotes delimit arguments; doubled quotes
// inside a quoted argument represent one literal quote. No expansion or evaluation.
func tokenize(line string) ([]string, error) {
	args := []string{}
	var b strings.Builder
	var quote rune
	started := false
	r := []rune(line)
	for i := 0; i < len(r); i++ {
		ch := r[i]
		if quote != 0 {
			if ch == quote {
				if i+1 < len(r) && r[i+1] == quote {
					b.WriteRune(ch)
					i++
				} else {
					quote = 0
				}
			} else {
				b.WriteRune(ch)
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			started = true
		} else if ch == ' ' || ch == '\t' {
			if started {
				args = append(args, b.String())
				b.Reset()
				started = false
			}
		} else {
			b.WriteRune(ch)
			started = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}
	if started {
		args = append(args, b.String())
	}
	return args, nil
}
func (a *App) shell(ctx context.Context, input *bufio.Reader, out io.Writer) int {
	fmt.Fprintln(out, "AslUnix", version, "— local inspection toolkit. Type help; exit to quit.")
	history := []string{}
	editor := newLineEditor(input, out)
	defer editor.Close()
	for ctx.Err() == nil {
		line, e := editor.ReadLine("aslunix> ", history)
		if e != nil {
			if e != io.EOF {
				fmt.Fprintln(out, e)
			}
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}
		if line == "history" {
			for i, l := range history {
				fmt.Fprintf(out, "%d  %s\n", i+1, safeText(l))
			}
			continue
		}
		history = append(history, line)
		if len(history) > 500 {
			history = history[1:]
		}
		args, e := tokenize(line)
		if e != nil {
			fmt.Fprintln(out, e)
			continue
		}
		a.execute(ctx, args, input, out)
	}
	return 0
}

var commandList = []string{"help", "health", "disks", "df", "raid", "resources", "services", "events", "watch", "report", "accounts", "admins", "permissions", "identity", "network", "ping", "traceroute", "dns", "tcp", "http", "tls", "proxy", "time", "domain", "capture", "pwd", "cd", "ls", "cat", "head", "tail", "grep", "find", "du", "cp", "mv", "mkdir", "rm", "follow", "compare", "sha256", "archive", "os", "updates", "drivers", "tasks", "hardware", "registry", "wsus", "iis", "dotnet", "signature", "tools", "vmware", "sql", "version", "history", "exit"}

func completions(line string) []string {
	if !strings.ContainsAny(line, " \t") {
		out := []string{}
		for _, n := range commandList {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				out = append(out, n)
			}
		}
		sort.Strings(out)
		return out
	}
	// Complete filesystem paths, preserving quoted Windows paths and preceding args.
	start := strings.LastIndexAny(line, " \t") + 1
	quote := byte(0)
	for i := 0; i < len(line); i++ {
		if line[i] == '"' || line[i] == '\'' {
			if quote == 0 {
				quote = line[i]
				start = i + 1
			} else if quote == line[i] {
				quote = 0
			}
		}
	}
	prefix := line[:start]
	partial := line[start:]
	matches, _ := filepath.Glob(partial + "*")
	out := []string{}
	for _, m := range matches {
		if strings.ContainsAny(m, " \t") && quote == 0 {
			out = append(out, prefix+`"`+m+`"`)
		} else if quote != 0 {
			out = append(out, prefix+m+string(quote))
		} else {
			out = append(out, prefix+m)
		}
	}
	return out
}
