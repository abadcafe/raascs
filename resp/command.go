package resp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type argReader interface {
	ArgCount() int
	ReadArg(count int) ([][]byte, error)
	DiscardAllArgs() error
}

type Command struct {
	name string
	w *bufio.Writer
	argReader
}

type CommandFlag struct {
	NeedValue bool
	ExclusiveFlag *bool
	Receiver  func(s []byte) error
}

var ErrNoMoreArguments = errors.New("no more arguments")

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

func readCommand(c *cliConn) (*Command, error) {
	r := bufio.NewReader(c)

	line, err := getLine(r)
	if err != nil {
		return nil, err
	} else if len(line) <= 0 {
		return nil, nil
	}

	var argReader argReader
	var name string
	if line[0] == '*' {
		argc, err := strconv.Atoi(string(line[1:]))
		if err != nil {
			return nil, err
		}

		buf, err := readBulkString(r)
		if err != nil {
			return nil, err
		}

		name = string(bytes.ToUpper(buf))
		argReader = &bulkStringArgReader{
			r:        r,
			argCount: argc - 1,
			argCur:   0,
		}
	} else if (line[0] >= 'a' && line[0] <= 'z') || (line[0] >= 'A' && line[0] <= 'Z') {
		ss := bytes.Fields(line)
		name = string(bytes.ToUpper(ss[0]))
		argReader = &simpleStringArgReader{
			args:   ss[1:],
			argCur: 0,
		}
	} else {
		return nil, fmt.Errorf("command string invalid")
	}

	return &Command{
		name:      name,
		w:         bufio.NewWriter(c),
		argReader: argReader,
	}, nil
}

func (c *Command) Name() string {
	return c.name
}

func (c *Command) ParseArgs(flags map[string]*CommandFlag) error {
	var valueReceiver func(s []byte) error = nil

	for c.ArgCount() > 0 {
		args, err := c.ReadArg(1)
		if err != nil {
			return nil
		}

		if valueReceiver == nil {
			arg := string(bytes.ToUpper(args[0]))
			cmdFlag, ok := flags[arg]
			if !ok {
				return fmt.Errorf("unexpected argument occurred: %s", arg)
			}

			if cmdFlag.ExclusiveFlag != nil {
				if *cmdFlag.ExclusiveFlag {
					return fmt.Errorf("argument %s can not occur with some other arguments at same time", arg)
				} else {
					*cmdFlag.ExclusiveFlag = true
				}
			}

			if !cmdFlag.NeedValue {
				err = cmdFlag.Receiver(nil)
				if err != nil {
					return err
				}
			} else {
				valueReceiver = cmdFlag.Receiver
			}
		} else {
			err := valueReceiver(args[0])
			if err != nil {
				return err
			}

			valueReceiver = nil
		}
	}

	if valueReceiver != nil {
		return fmt.Errorf("the last argument needs value but absent")
	}

	return nil
}

func (c *Command) WriteInt(i int) error {
	_, err := c.w.WriteString(fmt.Sprintf(":%d\r\n", i))
	return err
}

func (c *Command) WriteSimpleString(s string) error {
	_, err := c.w.WriteString(fmt.Sprintf("+%s\r\n", s))
	return err
}

func (c *Command) WriteBulkString(s []byte) error {
	_, err := c.w.WriteString(fmt.Sprintf("$%d\r\n", len(s)))
	if err != nil {
		return err
	}

	_, err = c.w.Write(s)
	if err != nil {
		return err
	}

	_, err = c.w.Write([]byte("\r\n"))
	if err != nil {
		return err
	}

	return err
}

func (c *Command) WriteNullBulkString() error {
	_, err := c.w.WriteString("$-1\r\n")
	return err
}

func (c *Command) WriteArrayLen(i int) error {
	_, err := c.w.WriteString(fmt.Sprintf("*%d\r\n", i))
	return err
}

func (c *Command) WriteNullArray() error {
	_, err := c.w.WriteString("$-1\r\n")
	return err
}

func (c *Command) WriteError(s string) error {
	err := c.DiscardAllArgs()
	if err != nil {
		return err
	}

	_, err = c.w.WriteString(fmt.Sprintf("-%s\r\n", s))
	return err
}

func (c *Command) FlushWrites() error {
	return c.w.Flush()
}
