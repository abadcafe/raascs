package resp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func getLine(r *bufio.Reader) ([]byte, error) {
	buf, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return bytes.TrimRight(buf, "\r\n"), nil
}

func readBulkString(r *bufio.Reader) ([]byte, error) {
	line, err := getLine(r)
	if err != nil {
		return nil, err
	}

	if line[0] != '$' {
		return nil, fmt.Errorf("prefix of bulk string error")
	}

	l, err := strconv.Atoi(string(line[1:]))
	if err != nil {
		return nil, err
	}

	buf := make([]byte, l)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	_, err = r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func readCommand(c *cliConn) (Command, error) {
	r := bufio.NewReader(c)

	line, err := getLine(r)
	if err != nil {
		return nil, err
	} else if len(line) <= 0 {
		return nil, nil
	}

	if line[0] == '*' {
		argc, err := strconv.Atoi(string(line[1:]))
		if err != nil {
			return nil, err
		}

		buf, err := readBulkString(r)
		if err != nil {
			return nil, err
		}

		return &normalCommand{
			commandWriter: commandWriter{
				w:    bufio.NewWriter(c),
				name: string(bytes.ToUpper(buf)),
			},
			r:        r,
			argCount: argc - 1,
		}, nil
	} else if (line[0] >= 'a' && line[0] <= 'z') || (line[0] >= 'A' && line[0] <= 'Z') {
		ss := strings.Fields(string(line))
		return &inlineCommand{
			commandWriter: commandWriter{
				name: strings.ToUpper(ss[0]),
				w:    bufio.NewWriter(c),
			},
			args:   ss[1:],
			argCur: 0,
		}, nil
	}

	return nil, fmt.Errorf("command string invalid")
}

type Command interface {
	Name() string
	ArgCount() int
	ReadArg() (string, error)
	WriteInt(i int) error
	WriteSimpleString(s string) error
	WriteBulkString(s string) error
	WriteArrayLen(i int) error
	WriteError(s string) error
	FlushWrites() error
}

type commandWriter struct {
	name string
	w    *bufio.Writer
}

func (c *commandWriter) Name() string {
	return c.name
}

func (c *commandWriter) WriteInt(i int) error {
	_, err := c.w.WriteString(fmt.Sprintf(":%d\r\n", i))
	if err != nil {
		return err
	}

	return nil
}

func (c *commandWriter) WriteSimpleString(s string) error {
	_, err := c.w.WriteString(fmt.Sprintf("+%s\r\n", s))
	if err != nil {
		return err
	}

	return nil
}

func (c *commandWriter) WriteBulkString(s string) error {
	_, err := c.w.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s))
	if err != nil {
		return err
	}

	return nil
}

func (c *commandWriter) WriteArrayLen(i int) error {
	_, err := c.w.WriteString(fmt.Sprintf("*%d\r\n", i))
	if err != nil {
		return err
	}

	return nil
}

func (c *commandWriter) WriteError(s string) error {
	_, err := c.w.WriteString(fmt.Sprintf("-%s\r\n", s))
	if err != nil {
		return err
	}

	return nil
}

func (c *commandWriter) FlushWrites() error {
	return c.w.Flush()
}
