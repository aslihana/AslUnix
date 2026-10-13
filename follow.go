package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func follow(ctx context.Context, args []string, out io.Writer) int {
	if len(args) != 1 && len(args) != 3 {
		return usageError(out, "follow PATH [--seconds N]")
	}
	if len(args) == 3 {
		n, e := strconv.Atoi(args[2])
		if args[1] != "--seconds" || e != nil || n < 1 || n > 3600 {
			return usageError(out, "follow --seconds must be 1..3600")
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(n)*time.Second)
		defer cancel()
	}
	f, e := os.Open(args[0])
	if e != nil {
		return usageError(out, e.Error())
	}
	defer func() { f.Close() }()
	info, e := f.Stat()
	if e != nil {
		return usageError(out, e.Error())
	}
	if !info.Mode().IsRegular() {
		return usageError(out, "follow requires a regular file")
	}
	offset, e := f.Seek(0, io.SeekEnd)
	if e != nil {
		return usageError(out, e.Error())
	}
	fmt.Fprintln(out, "Following from current end; Ctrl+C stops.")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	buf := make([]byte, 64<<10)
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
		}
		current, e := os.Stat(args[0])
		if e != nil {
			fmt.Fprintln(out, "Waiting for log:", e)
			continue
		}
		if !os.SameFile(info, current) {
			newFile, e := os.Open(args[0])
			if e != nil {
				continue
			}
			f.Close()
			f = newFile
			info = current
			offset = 0
		}
		if current.Size() < offset {
			if _, e = f.Seek(0, io.SeekStart); e != nil {
				return usageError(out, e.Error())
			}
			offset = 0
		}
		// Read at most 1 MiB per tick so cancellation remains responsive for busy logs.
		for i := 0; i < 16; i++ {
			n, e := f.Read(buf)
			if n > 0 {
				fmt.Fprint(out, safeText(string(buf[:n])))
				offset += int64(n)
			}
			if e == io.EOF {
				break
			}
			if e != nil {
				return usageError(out, e.Error())
			}
		}
	}
}
