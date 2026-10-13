//go:build !windows

package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type lineEditor struct {
	in  *bufio.Reader
	out io.Writer
}

func newLineEditor(in *bufio.Reader, out io.Writer) *lineEditor { return &lineEditor{in, out} }
func (e *lineEditor) Close()                                    {}
func (e *lineEditor) ReadLine(prompt string, history []string) (string, error) {
	fmt.Fprint(e.out, prompt)
	s, err := e.in.ReadString('\n')
	if len(s) > 0 {
		return strings.TrimRight(s, "\r\n"), nil
	}
	return s, err
}
