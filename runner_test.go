package main

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestCapturedOutputBounded(t *testing.T) {
	b := cappedBuffer{max: 4}
	n, e := b.Write([]byte("abcdefgh"))
	if n != 8 || e != nil || b.String() != "abcd" {
		t.Fatal(n, e, b.String())
	}
	b.Write([]byte("ijk"))
	if b.Len() != 4 {
		t.Fatal("cap grew")
	}
}
func TestSubprocessDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows process-tree timeout is covered by Windows smoke validation")
	}
	path, e := exec.LookPath("sh")
	if e != nil {
		t.Skip(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, e = runBounded(ctx, path, []string{"-c", "exec sleep 10"}, nil)
	if e != context.DeadlineExceeded {
		t.Fatal(e)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatal("deadline not bounded")
	}
}
