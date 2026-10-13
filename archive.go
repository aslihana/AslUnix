package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func archiveTarget(root, name string) (string, error) {
	// Reject both Windows and Unix traversal syntax even when tests run on another OS.
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("unsafe archive traversal %q", name)
		}
	}
	path := filepath.Join(root, filepath.FromSlash(name))
	rel, e := filepath.Rel(root, path)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive path escapes destination")
	}
	return path, nil
}
func rejectSymlinkParents(path string) error {
	abs, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	for {
		info, e := os.Lstat(abs)
		if e == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path component: %s", abs)
		}
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		next := filepath.Dir(abs)
		if next == abs {
			break
		}
		abs = next
	}
	return nil
}
func doArchive(ctx context.Context, o fileOpts, r *Result) error {
	if len(o.args) < 2 {
		return fmt.Errorf("archive list FILE.zip | archive extract FILE.zip DEST [--dry-run]")
	}
	action := o.args[0]
	if action != "list" && action != "extract" {
		return fmt.Errorf("archive supports ZIP list/extract")
	}
	expected := 2
	if action == "extract" {
		expected = 3
	}
	if e := requireArgs(o.args, expected); e != nil {
		return e
	}
	z, e := zip.OpenReader(o.args[1])
	if e != nil {
		return e
	}
	defer z.Close()
	if len(z.File) > 10000 {
		return fmt.Errorf("archive exceeds 10000 entries")
	}
	rows := []map[string]any{}
	var total uint64
	root := ""
	if action == "extract" {
		root, e = filepath.Abs(o.args[2])
		if e != nil {
			return e
		}
		if e = rejectSymlinkParents(root); e != nil {
			return e
		}
	}
	for _, f := range z.File {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rows = append(rows, map[string]any{"Name": f.Name, "Bytes": f.UncompressedSize64, "CompressedBytes": f.CompressedSize64})
		if f.UncompressedSize64 > 1<<30 || total > 1<<30-f.UncompressedSize64 {
			return fmt.Errorf("archive exceeds 1 GiB uncompressed limit")
		}
		total += f.UncompressedSize64
		if action == "extract" {
			if f.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink archive entries are unsupported")
			}
			target, e := archiveTarget(root, f.Name)
			if e != nil {
				return e
			}
			if e = rejectSymlinkParents(target); e != nil {
				return e
			}
			if !f.FileInfo().IsDir() {
				if _, e = os.Lstat(target); e == nil {
					return fmt.Errorf("refusing to overwrite %s", target)
				} else if !os.IsNotExist(e) {
					return e
				}
			}
		}
	}
	r.Data = rows
	if action == "list" {
		r.Summary = "ZIP contents"
		return nil
	}
	r.Summary = "Would extract ZIP to " + root
	if o.dry {
		return nil
	}
	for _, f := range z.File {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		target, _ := archiveTarget(root, f.Name)
		if e = rejectSymlinkParents(target); e != nil {
			return e
		}
		if f.FileInfo().IsDir() {
			if e = os.MkdirAll(target, 0755); e != nil {
				return e
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			return e
		}
		in, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			in.Close()
			return e
		}
		n, e := copyContext(ctx, out, io.LimitReader(in, int64(f.UncompressedSize64)+1))
		in.Close()
		ce := out.Close()
		if e != nil || ce != nil || uint64(n) != f.UncompressedSize64 {
			os.Remove(target)
			return fmt.Errorf("archive extraction failed for %s: copy=%v close=%v bytes=%d", f.Name, e, ce, n)
		}
	}
	r.Summary = "Extracted ZIP to " + root
	r.Limitations = []string{"Extraction is sequential; on failure completed files remain. No ACL or ownership restoration. Extract only into a directory not writable by untrusted concurrent processes."}
	return nil
}
