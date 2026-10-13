package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fileRun(t *testing.T, name string, args []string, answer string) Result {
	t.Helper()
	a := testApp(fakeRunner{})
	var out bytes.Buffer
	return a.fileCommand(context.Background(), name, args, bufio.NewReader(strings.NewReader(answer)), &out)
}
func TestRemoveConfirmationAndPreview(t *testing.T) {
	p := filepath.Join(t.TempDir(), "file")
	os.WriteFile(p, []byte("retain"), 0600)
	r := fileRun(t, "rm", []string{p, "--dry-run"}, "yes\n")
	if r.Status != OK {
		t.Fatal(r)
	}
	if _, e := os.Stat(p); e != nil {
		t.Fatal("dry run removed file")
	}
	r = fileRun(t, "rm", []string{p}, "no\n")
	if r.Status != NotApplicable {
		t.Fatal(r)
	}
	if _, e := os.Stat(p); e != nil {
		t.Fatal("unconfirmed deletion")
	}
	r = fileRun(t, "rm", []string{p}, "yes\n")
	if r.Status != OK {
		t.Fatal(r)
	}
	if _, e := os.Stat(p); !os.IsNotExist(e) {
		t.Fatal("confirmed deletion failed")
	}
}
func TestRootAndCWDRemovalRejected(t *testing.T) {
	cwd, _ := os.Getwd()
	for _, p := range []string{cwd, filepath.Dir(cwd)} {
		r := fileRun(t, "rm", []string{p, "--recursive", "--yes", "--dry-run"}, "")
		if r.Status != Unknown {
			t.Fatal("dangerous remove accepted", p)
		}
	}
}
func TestCopyNoSilentOverwrite(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	os.WriteFile(a, []byte("new"), 0600)
	os.WriteFile(b, []byte("old"), 0600)
	r := fileRun(t, "cp", []string{a, b}, "no\n")
	if r.Status != NotApplicable {
		t.Fatal(r)
	}
	data, _ := os.ReadFile(b)
	if string(data) != "old" {
		t.Fatal("silently replaced")
	}
	r = fileRun(t, "cp", []string{a, b}, "yes\n")
	if r.Status != OK {
		t.Fatal(r)
	}
	data, _ = os.ReadFile(b)
	if string(data) != "new" {
		t.Fatal("copy failed")
	}
	if r = fileRun(t, "cp", []string{a, a, "--yes"}, ""); r.Status != Unknown {
		t.Fatal("self copy allowed")
	}
}
func TestArchiveTraversal(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"../escape", "a/../../escape", `..\escape`, `C:\Windows\evil`, "/tmp/evil", `\\server\evil`} {
		if _, e := archiveTarget(root, name); e == nil {
			t.Fatal("unsafe path", name)
		}
	}
	if _, e := archiveTarget(root, "safe/path.txt"); e != nil {
		t.Fatal(e)
	}
}
func writeZip(t *testing.T, path, name string) {
	t.Helper()
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	w := zip.NewWriter(f)
	entry, e := w.Create(name)
	if e != nil {
		t.Fatal(e)
	}
	entry.Write([]byte("payload"))
	w.Close()
	f.Close()
}
func TestArchiveExtractionRefusesEscapeAndOverwrite(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	z := filepath.Join(dir, "a.zip")
	dest := filepath.Join(dir, "extract")
	writeZip(t, z, "../escape.txt")
	r := fileRun(t, "archive", []string{"extract", z, dest}, "")
	if r.Status != Unknown {
		t.Fatal(r)
	}
	if _, e := os.Stat(filepath.Join(dir, "escape.txt")); !os.IsNotExist(e) {
		t.Fatal("escape exists")
	}
	writeZip(t, z, "safe.txt")
	r = fileRun(t, "archive", []string{"extract", z, dest, "--dry-run"}, "")
	if r.Status != OK {
		t.Fatal(r)
	}
	if _, e := os.Stat(dest); !os.IsNotExist(e) {
		t.Fatal("preview mutated directory")
	}
	r = fileRun(t, "archive", []string{"extract", z, dest}, "")
	if r.Status != OK {
		t.Fatal(r)
	}
	r = fileRun(t, "archive", []string{"extract", z, dest}, "")
	if r.Status != Unknown {
		t.Fatal("overwrite accepted")
	}
}
func TestArchiveSymlinkParent(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if e := os.Symlink(t.TempDir(), link); e != nil {
		t.Skip(e)
	}
	if e := rejectSymlinkParents(filepath.Join(link, "file")); e == nil {
		t.Fatal("symlink accepted")
	}
}
func TestCopyCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if _, e := copyContext(ctx, &out, strings.NewReader("data")); e != context.Canceled {
		t.Fatal(e)
	}
	if out.Len() != 0 {
		t.Fatal("copied after cancellation")
	}
}
