//go:build windows

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var kernel = syscall.NewLazyDLL("kernel32.dll")
var getMode = kernel.NewProc("GetConsoleMode")
var setMode = kernel.NewProc("SetConsoleMode")
var readInput = kernel.NewProc("ReadConsoleInputW")

type lineEditor struct {
	in     *bufio.Reader
	out    io.Writer
	handle syscall.Handle
	mode   uint32
	native bool
}

func newLineEditor(in *bufio.Reader, out io.Writer) *lineEditor {
	e := &lineEditor{in: in, out: out, handle: syscall.Handle(os.Stdin.Fd())}
	ok, _, _ := getMode.Call(uintptr(e.handle), uintptr(unsafe.Pointer(&e.mode)))
	// Use raw input only for a real console. Leave redirected stdin line-oriented.
	if ok != 0 {
		e.native = true
	}
	return e
}
func (e *lineEditor) Close() {
	if e.native {
		setMode.Call(uintptr(e.handle), uintptr(e.mode))
	}
}
func (e *lineEditor) ReadLine(prompt string, history []string) (string, error) {
	fmt.Fprint(e.out, prompt)
	if !e.native {
		s, err := e.in.ReadString('\n')
		if len(s) > 0 {
			return strings.TrimRight(s, "\r\n"), nil
		}
		return s, err
	}
	// Raw mode applies only during editing; confirmations use normal console input.
	setMode.Call(uintptr(e.handle), uintptr(e.mode&^uint32(2|4|1)))
	defer setMode.Call(uintptr(e.handle), uintptr(e.mode))
	line := []rune{}
	cursor := 0
	hi := len(history)
	draft := ""
	lastWidth := 0
	redraw := func() {
		s := safeText(string(line))
		fmt.Fprint(e.out, "\r", prompt, s, strings.Repeat(" ", max(0, lastWidth-len(line))), "\r", prompt, safeText(string(line[:cursor])))
		lastWidth = len(line)
	}
	for {
		var event [20]byte
		var read uint32
		ok, _, err := readInput.Call(uintptr(e.handle), uintptr(unsafe.Pointer(&event[0])), 1, uintptr(unsafe.Pointer(&read)))
		if ok == 0 {
			return "", err
		}
		if read == 0 || *(*uint16)(unsafe.Pointer(&event[0])) != 1 || *(*uint32)(unsafe.Pointer(&event[4])) == 0 {
			continue
		}
		key := *(*uint16)(unsafe.Pointer(&event[10]))
		ch := *(*uint16)(unsafe.Pointer(&event[14]))
		switch key {
		case 13:
			fmt.Fprintln(e.out)
			return string(line), nil
		case 8:
			if cursor > 0 {
				line = append(line[:cursor-1], line[cursor:]...)
				cursor--
			}
		case 46:
			if cursor < len(line) {
				line = append(line[:cursor], line[cursor+1:]...)
			}
		case 37:
			if cursor > 0 {
				cursor--
			}
		case 39:
			if cursor < len(line) {
				cursor++
			}
		case 36:
			cursor = 0
		case 35:
			cursor = len(line)
		case 38:
			if hi > 0 {
				if hi == len(history) {
					draft = string(line)
				}
				hi--
				line = []rune(history[hi])
				cursor = len(line)
			}
		case 40:
			if hi < len(history) {
				hi++
				if hi == len(history) {
					line = []rune(draft)
				} else {
					line = []rune(history[hi])
				}
				cursor = len(line)
			}
		case 9:
			matches := completions(string(line))
			if len(matches) == 1 {
				line = []rune(matches[0])
				cursor = len(line)
			} else if len(matches) > 1 {
				fmt.Fprintln(e.out, "\n"+strings.Join(matches, "  "))
				lastWidth = 0
			}
		default:
			if ch == 3 {
				fmt.Fprintln(e.out, "^C")
				return "", nil
			}
			if ch == 4 && len(line) == 0 {
				return "", io.EOF
			}
			if ch >= 32 {
				line = append(line, 0)
				copy(line[cursor+1:], line[cursor:])
				line[cursor] = rune(ch)
				cursor++
			}
		}
		redraw()
	}
}
