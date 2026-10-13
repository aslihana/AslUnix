package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"
)

//go:embed collectors/*.ps1
var scripts embed.FS

type Runner interface {
	Collect(context.Context, string, map[string]any) (json.RawMessage, string, []string, error)
}
type PowerShellRunner struct{}

func (PowerShellRunner) Collect(ctx context.Context, name string, input map[string]any) (json.RawMessage, string, []string, error) {
	if runtime.GOOS != "windows" {
		return nil, "", nil, fmt.Errorf("Windows collector unavailable on %s; run this executable on Windows", runtime.GOOS)
	}
	script, e := scripts.ReadFile("collectors/" + name + ".ps1")
	if e != nil {
		return nil, "", nil, e
	}
	// Use the inbox 64-bit executable, not a PATH-supplied PowerShell.
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	payload, e := json.Marshal(input)
	if e != nil {
		return nil, "", nil, e
	}
	preamble := `$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; [Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false); $p=$env:ASLUNIX_INPUT | ConvertFrom-Json; $limitations=@(); `
	wrapped := preamble + string(script) + `; [pscustomobject]@{data=@($data);limitations=@($limitations)} | ConvertTo-Json -Depth 12 -Compress`
	u := utf16.Encode([]rune(wrapped))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	out, rawErr, e := runBounded(ctx, ps, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(b)}, append(os.Environ(), "ASLUNIX_INPUT="+string(payload)))
	raw := strings.TrimSpace(string(out))
	if rawErr != "" {
		raw += "\nSTDERR: " + rawErr
	}
	if e != nil {
		return nil, raw, nil, e
	}
	var envelope struct {
		Data        json.RawMessage `json:"data"`
		Limitations []string        `json:"limitations"`
	}
	if e = json.Unmarshal(bytes.TrimPrefix(out, []byte{0xef, 0xbb, 0xbf}), &envelope); e != nil {
		return nil, raw, nil, fmt.Errorf("invalid collector JSON: %w", e)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, raw, envelope.Limitations, fmt.Errorf("collector did not return a data array")
	}
	return envelope.Data, raw, envelope.Limitations, nil
}

// Bound captured output; a noisy collector must not exhaust an OT server's memory.
type cappedBuffer struct {
	bytes.Buffer
	max int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.max - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		b.Buffer.Write(p)
	}
	return n, nil
}
func runBounded(ctx context.Context, path string, args, env []string) ([]byte, string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = env
	cmd.WaitDelay = 2 * time.Second
	stdout := &cappedBuffer{max: 8 << 20}
	stderr := &cappedBuffer{max: 1 << 20}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	configureProcess(cmd)
	e := cmd.Run()
	if ctx.Err() != nil {
		e = ctx.Err()
	}
	if (stdout.Len() >= stdout.max || stderr.Len() >= stderr.max) && e == nil {
		e = fmt.Errorf("collector output reached stdout/stderr capture limit")
	}
	return stdout.Bytes(), strings.TrimSpace(stderr.String()), e
}
