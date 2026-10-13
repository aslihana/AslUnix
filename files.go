package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var fileCommands = map[string]bool{"pwd": true, "cd": true, "ls": true, "cat": true, "head": true, "tail": true, "grep": true, "find": true, "du": true, "cp": true, "mv": true, "mkdir": true, "rm": true, "follow": true, "compare": true, "sha256": true, "archive": true}

func isFileCommand(name string) bool { return fileCommands[name] }

type fileOpts struct {
	dry, yes, recursive bool
	n                   int
	args                []string
}

func parseFileOpts(args []string) (fileOpts, error) {
	o := fileOpts{n: 10}
	literal := false
	for i := 0; i < len(args); i++ {
		s := args[i]
		if literal {
			o.args = append(o.args, s)
			continue
		}
		switch s {
		case "--":
			literal = true
		case "--dry-run":
			o.dry = true
		case "--yes":
			o.yes = true
		case "--recursive":
			o.recursive = true
		case "-n":
			if i+1 >= len(args) {
				return o, fmt.Errorf("-n requires a positive number")
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 || n > 100000 {
				return o, fmt.Errorf("-n must be 1..100000")
			}
			o.n = n
		default:
			if strings.HasPrefix(s, "--") {
				return o, fmt.Errorf("unknown option %s; use -- before a filename beginning with --", s)
			}
			o.args = append(o.args, s)
		}
	}
	return o, nil
}
func (a *App) fileCommand(ctx context.Context, name string, args []string, input *bufio.Reader, out io.Writer) Result {
	r := result(name, OK, "Go filesystem", "Completed", nil)
	o, e := parseFileOpts(args)
	if e != nil {
		r.Status = Unknown
		r.Summary = e.Error()
		return r
	}
	if o.yes && name != "rm" && name != "cp" {
		r.Status = Unknown
		r.Summary = "--yes is supported only by rm/cp"
		return r
	}
	if o.recursive && name != "rm" {
		r.Status = Unknown
		r.Summary = "--recursive is supported only by rm"
		return r
	}
	if o.dry && !contains([]string{"cd", "cp", "mv", "mkdir", "rm", "archive"}, name) {
		r.Status = Unknown
		r.Summary = "--dry-run is not supported by " + name
		return r
	}
	for _, arg := range args {
		if arg == "-n" && name != "head" && name != "tail" {
			r.Status = Unknown
			r.Summary = "-n is supported only by head/tail"
			return r
		}
	}
	e = a.doFile(ctx, name, o, input, out, &r)
	if e != nil {
		r.Status = Unknown
		r.Summary = e.Error()
	}
	return r
}
func requireArgs(args []string, n int) error {
	if len(args) != n {
		return fmt.Errorf("expected %d arguments, received %d; use help for syntax", n, len(args))
	}
	return nil
}
func (a *App) doFile(ctx context.Context, name string, o fileOpts, input *bufio.Reader, out io.Writer, r *Result) error {
	p := o.args
	switch name {
	case "pwd":
		if e := requireArgs(p, 0); e != nil {
			return e
		}
		path, e := os.Getwd()
		r.Data = path
		return e
	case "cd":
		if e := requireArgs(p, 1); e != nil {
			return e
		}
		if o.dry {
			r.Data = p[0]
			return nil
		}
		return os.Chdir(p[0])
	case "ls":
		if len(p) > 1 {
			return requireArgs(p, 1)
		}
		path := "."
		if len(p) == 1 {
			path = p[0]
		}
		entries, e := os.ReadDir(path)
		if e != nil {
			return e
		}
		rows := []map[string]any{}
		for _, v := range entries {
			info, e := v.Info()
			if e != nil {
				return e
			}
			rows = append(rows, map[string]any{"Name": v.Name(), "Bytes": info.Size(), "Directory": v.IsDir(), "Modified": info.ModTime().Format(time.RFC3339), "Mode": info.Mode().String()})
		}
		r.Data = rows
	case "cat", "head", "tail", "grep":
		expected := 1
		if name == "grep" {
			expected = 2
		}
		if e := requireArgs(p, expected); e != nil {
			return e
		}
		path := p[len(p)-1]
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		if name == "cat" {
			b, e := readLimited(f, 8<<20)
			if e != nil {
				return e
			}
			r.Data = string(b)
			return nil
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 4096), 1<<20)
		lines := []string{}
		ring := make([]string, o.n)
		count := 0
		total := 0
		for scanner.Scan() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			line := scanner.Text()
			total += len(line)
			if total > 64<<20 {
				return fmt.Errorf("scan exceeds 64 MiB; use a smaller file or follow")
			}
			if name == "head" {
				lines = append(lines, line)
				if len(lines) == o.n {
					break
				}
			} else if name == "tail" {
				ring[count%o.n] = line
				count++
			} else if strings.Contains(line, p[0]) {
				lines = append(lines, fmt.Sprintf("%d:%s", count+1, line))
				count++
			} else {
				count++
			}
		}
		if e = scanner.Err(); e != nil {
			return e
		}
		if name == "tail" {
			start := max(0, count-o.n)
			for i := start; i < count; i++ {
				lines = append(lines, ring[i%o.n])
			}
		}
		r.Data = strings.Join(lines, "\n")
	case "find", "du":
		if len(p) < 1 || len(p) > 2 {
			return fmt.Errorf("%s PATH [NAME_GLOB]", name)
		}
		pattern := "*"
		if len(p) == 2 {
			if name != "find" {
				return fmt.Errorf("du accepts one path")
			}
			pattern = p[1]
			if _, e := filepath.Match(pattern, ""); e != nil {
				return e
			}
		}
		rows := []map[string]any{}
		var total int64
		count := 0
		e := filepath.WalkDir(p[0], func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			count++
			if count > 100000 {
				return fmt.Errorf("walk reached 100000 entries; select a narrower path")
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if name == "find" {
				match, _ := filepath.Match(pattern, d.Name())
				if match {
					rows = append(rows, map[string]any{"Path": path, "Directory": d.IsDir()})
				}
			} else if !d.IsDir() {
				info, e := d.Info()
				if e != nil {
					return e
				}
				total += info.Size()
			}
			return nil
		})
		if e != nil {
			return e
		}
		if name == "find" {
			r.Data = rows
		} else {
			r.Data = map[string]any{"Path": p[0], "Bytes": total, "GiB": float64(total) / (1 << 30), "Entries": count}
			r.Limitations = []string{"Logical file sizes; hard links count per path; symlinks skipped; not allocated disk usage"}
		}
	case "sha256":
		if e := requireArgs(p, 1); e != nil {
			return e
		}
		f, e := os.Open(p[0])
		if e != nil {
			return e
		}
		defer f.Close()
		h := sha256.New()
		if _, e = copyContext(ctx, h, f); e != nil {
			return e
		}
		r.Data = map[string]any{"Path": p[0], "SHA256": hex.EncodeToString(h.Sum(nil))}
	case "compare":
		if e := requireArgs(p, 2); e != nil {
			return e
		}
		lf, e := os.Open(p[0])
		if e != nil {
			return e
		}
		defer lf.Close()
		left, e := readLimited(lf, 8<<20)
		if e != nil {
			return e
		}
		rf, e := os.Open(p[1])
		if e != nil {
			return e
		}
		defer rf.Close()
		right, e := readLimited(rf, 8<<20)
		if e != nil {
			return e
		}
		if len(left) > 8<<20 || len(right) > 8<<20 {
			return fmt.Errorf("compare file exceeds 8 MiB")
		}
		l := strings.Split(strings.ReplaceAll(string(left), "\r\n", "\n"), "\n")
		rr := strings.Split(strings.ReplaceAll(string(right), "\r\n", "\n"), "\n")
		rows := []map[string]any{}
		for i := 0; i < max(len(l), len(rr)); i++ {
			ls, rs := "<absent>", "<absent>"
			if i < len(l) {
				ls = l[i]
			}
			if i < len(rr) {
				rs = rr[i]
			}
			if ls != rs {
				rows = append(rows, map[string]any{"Line": i + 1, "Left": ls, "Right": rs})
			}
		}
		r.Data = rows
		r.Summary = fmt.Sprintf("%d differing line positions (not a sequence diff)", len(rows))
	case "mkdir":
		if e := requireArgs(p, 1); e != nil {
			return e
		}
		if o.dry {
			r.Summary = "Would create directory " + p[0]
			return nil
		}
		return os.Mkdir(p[0], 0755)
	case "cp", "mv":
		if e := requireArgs(p, 2); e != nil {
			return e
		}
		src, e := os.Lstat(p[0])
		if e != nil {
			return e
		}
		if !src.Mode().IsRegular() {
			return fmt.Errorf("cp/mv support regular files only; symlinks and directories are rejected")
		}
		if samePath(p[0], p[1]) {
			return fmt.Errorf("source and destination are the same path")
		}
		dest, e := os.Lstat(p[1])
		exists := e == nil
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if exists && (dest.Mode()&os.ModeSymlink != 0 || !dest.Mode().IsRegular()) {
			return fmt.Errorf("destination must be a regular file")
		}
		if exists && os.SameFile(src, dest) {
			return fmt.Errorf("source and destination identify the same file")
		}
		r.Summary = fmt.Sprintf("%s %q to %q", name, p[0], p[1])
		if o.dry {
			r.Summary = "Would " + r.Summary
			return nil
		}
		if exists && !confirm(input, out, o.yes, "overwrite "+p[1]) {
			r.Status = NotApplicable
			r.Summary = "Cancelled"
			return nil
		}
		if name == "mv" {
			if exists {
				return fmt.Errorf("mv does not replace existing files; choose an unused destination")
			}
			return moveFile(p[0], p[1])
		}
		return copyFile(ctx, p[0], p[1], exists, src.Mode())
	case "rm":
		if e := requireArgs(p, 1); e != nil {
			return e
		}
		abs, e := filepath.Abs(p[0])
		if e != nil {
			return e
		}
		cwd, e := os.Getwd()
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(abs, cwd)
		if e == nil && (rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")) {
			return fmt.Errorf("refusing to remove working directory or its ancestor")
		}
		if filepath.Dir(abs) == abs {
			return fmt.Errorf("refusing filesystem root")
		}
		info, e := os.Lstat(abs)
		if e != nil {
			return e
		}
		if info.IsDir() && !o.recursive {
			return fmt.Errorf("directory removal requires --recursive")
		}
		r.Summary = "Would remove " + abs
		if o.dry {
			return nil
		}
		if !confirm(input, out, o.yes, "remove "+abs) {
			r.Status = NotApplicable
			r.Summary = "Cancelled"
			return nil
		}
		if info.IsDir() {
			e = os.RemoveAll(abs)
		} else {
			e = os.Remove(abs)
		}
		r.Summary = "Removed " + abs
		return e
	case "archive":
		return doArchive(ctx, o, r)
	case "follow":
		return fmt.Errorf("use follow PATH through the streaming command handler")
	}
	return nil
}
func samePath(a, b string) bool { aa, _ := filepath.Abs(a); bb, _ := filepath.Abs(b); return aa == bb }
func confirm(in *bufio.Reader, out io.Writer, yes bool, action string) bool {
	if yes {
		return true
	}
	fmt.Fprintf(out, "Confirm %s [type yes]: ", safeText(action))
	s, e := in.ReadString('\n')
	return e == nil && strings.TrimSpace(s) == "yes"
}
func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if e == nil && int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d byte limit", maxBytes)
	}
	return b, e
}
func copyContext(ctx context.Context, w io.Writer, r io.Reader) (int64, error) {
	buf := make([]byte, 64<<10)
	var total int64
	for {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		n, e := r.Read(buf)
		if n > 0 {
			wn, we := w.Write(buf[:n])
			total += int64(wn)
			if we != nil {
				return total, we
			}
			if wn != n {
				return total, io.ErrShortWrite
			}
		}
		if e == io.EOF {
			return total, nil
		}
		if e != nil {
			return total, e
		}
	}
}
func copyFile(ctx context.Context, src, dest string, replace bool, mode fs.FileMode) error {
	// Stage in the destination directory so interrupted copies never destroy the old file.
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	tmp, e := os.CreateTemp(filepath.Dir(dest), ".aslunix-copy-*")
	if e != nil {
		return e
	}
	defer os.Remove(tmp.Name())
	_, e = copyContext(ctx, tmp, in)
	ce := tmp.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if e = os.Chmod(tmp.Name(), mode.Perm()); e != nil {
		return e
	}
	if replace {
		return os.Rename(tmp.Name(), dest)
	}
	// Exclusive create prevents races that could overwrite a newly created target.
	target, e := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if e != nil {
		return e
	}
	stage, e := os.Open(tmp.Name())
	if e != nil {
		target.Close()
		os.Remove(dest)
		return e
	}
	_, e = copyContext(ctx, target, stage)
	stage.Close()
	ce = target.Close()
	if e != nil || ce != nil {
		os.Remove(dest)
		if e != nil {
			return e
		}
		return ce
	}
	return nil
}

// An exclusive hard-link creation prevents mv from racing into an overwrite.
// NTFS and same-volume regular files only; deletion failure rolls back the link.
func moveFile(src, dest string) error {
	if e := os.Link(src, dest); e != nil {
		return fmt.Errorf("mv requires same-volume hard-link support (NTFS): %w", e)
	}
	if e := os.Remove(src); e != nil {
		os.Remove(dest)
		return e
	}
	return nil
}
