package gogh

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirkon/errors"
	"github.com/sirkon/message"
)

// GoFmt formats source code with gofmt
func GoFmt(src []byte) ([]byte, error) {
	return runCommand(src, "gofmt")
}

// FancyFmt formats source code with github.com/sirkon/fancyfmt
func FancyFmt(src []byte) ([]byte, error) {
	return runCommand(src, "fancyfmt", "-")
}

func runCommand(input []byte, cmd string, params ...string) ([]byte, error) {
	var dest bytes.Buffer
	var errData bytes.Buffer
	c := exec.Command(cmd, params...)
	c.Stdin = bytes.NewReader(input)
	c.Stdout = &dest
	c.Stderr = &errData
	err := c.Run()
	if err == nil {
		return dest.Bytes(), nil
	}

	lines := bytes.Split(input, []byte{'\n'})
	p := newNumberPrinter(len(lines))
	for i, line := range lines {
		_, _ = os.Stdout.WriteString(p.num(i))
		_, _ = os.Stdout.WriteString(": ")
		_, _ = os.Stdout.Write(line)
		_, _ = os.Stdout.WriteString("\n")
	}
	_, _ = io.Copy(os.Stderr, &errData)
	message.Error(err)

	return nil, errors.New("failed to format")
}

type numberPrinter struct {
	digits int
}

func newNumberPrinter(total int) *numberPrinter {
	return &numberPrinter{
		digits: len(strconv.Itoa(total)),
	}
}

func (p numberPrinter) num(i int) string {
	num := strconv.Itoa(i + 1)
	if len(num) == p.digits {
		return num
	}

	return strings.Repeat("0", p.digits-len(num)) + num
}
