package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"
)

func safeText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
func render(out io.Writer, r Result, color bool) {
	status := string(r.Status)
	if color {
		code := "36"
		switch r.Status {
		case OK:
			code = "32"
		case Warning:
			code = "33"
		case Critical:
			code = "31"
		}
		status = "\x1b[" + code + "m" + status + "\x1b[0m"
	}
	fmt.Fprintf(out, "[%s] %s — %s\n  %s | %s\n", status, safeText(r.Check), safeText(r.Summary), r.Time.Format("2006-01-02T15:04:05Z07:00"), safeText(r.Source))
	if r.Data != nil {
		b, e := json.Marshal(r.Data)
		if e == nil {
			var data any
			if json.Unmarshal(b, &data) == nil {
				renderData(out, data)
			}
		}
	}

	for _, l := range r.Limitations {
		fmt.Fprintln(out, "  Limitation:", safeText(l))
	}
}
func table(out io.Writer, rows []map[string]any) {
	keys := []string{}
	seen := map[string]bool{}
	for _, r := range rows {
		for k := range r {
			if !seen[k] {
				keys = append(keys, k)
				seen[k] = true
			}
		}
	}
	sort.Strings(keys)
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(keys, "\t"))
	for _, r := range rows {
		cells := []string{}
		for _, k := range keys {
			v := r[k]
			s := ""
			switch t := v.(type) {
			case nil:
				s = "—"
			case float64:
				s = fmt.Sprintf("%.2f", t)
			case string:
				s = t
			default:
				b, _ := json.Marshal(v)
				s = string(b)
			}
			s = strings.ReplaceAll(strings.ReplaceAll(safeText(s), "\n", " | "), "\t", " ")
			cells = append(cells, s)
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}
	w.Flush()
}

func renderData(out io.Writer, data any) {
	switch v := data.(type) {
	case []any:
		if len(v) == 0 {
			fmt.Fprintln(out, "  (no rows)")
			return
		}
		rows := []map[string]any{}
		flat := true
		for _, item := range v {
			row, ok := item.(map[string]any)
			if !ok {
				flat = false
				break
			}
			for _, value := range row {
				switch value.(type) {
				case []any, map[string]any:
					flat = false
				}
			}
			rows = append(rows, row)
		}
		if flat {
			table(out, rows)
			return
		}
		for i, item := range v {
			if len(v) > 1 {
				fmt.Fprintf(out, "  Record %d:\n", i+1)
			}
			renderData(out, item)
		}
	case map[string]any:
		keys := []string{}
		scalars := map[string]any{}
		for k, value := range v {
			switch value.(type) {
			case []any, map[string]any:
				keys = append(keys, k)
			default:
				scalars[k] = value
			}
		}
		if len(scalars) > 0 {
			table(out, []map[string]any{scalars})
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(out, "  %s:\n", safeText(k))
			renderData(out, v[k])
		}
	case string:
		fmt.Fprintln(out, safeText(v))
	default:
		b, _ := json.Marshal(v)
		fmt.Fprintln(out, safeText(string(b)))
	}
}
