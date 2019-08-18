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

type CommandRequest struct {
	cliId int
	name  string
	w     *bufio.Writer
	argReader
}

type CommandFlag struct {
	NeedValue     bool
	ExclusiveFlag *bool
	ValueReceiver func(s []byte) error
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

func buildCommandRequest(c *cliConn) (*CommandRequest, error) {
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

	return &CommandRequest{
		cliId:     c.id,
		name:      name,
		w:         bufio.NewWriter(c),
		argReader: argReader,
	}, nil
}

func (r *CommandRequest) Name() string {
	return r.name
}

func (r *CommandRequest) ParseFlags(flags map[string]*CommandFlag) error {
	var valueReceiver func(s []byte) error = nil

	for r.ArgCount() > 0 {
		args, err := r.ReadArg(1)
		if err != nil {
			return nil
		}

		if valueReceiver == nil {
			arg := string(bytes.ToUpper(args[0]))
			cmdFlag, ok := flags[arg]
			if !ok {
				return fmt.Errorf("unexpected command flag occurred: %s", arg)
			}

			if cmdFlag.ExclusiveFlag != nil {
				if *cmdFlag.ExclusiveFlag {
					return fmt.Errorf("argument %s can not occur with some other arguments at same time", arg)
				} else {
					*cmdFlag.ExclusiveFlag = true
				}
			}

			if !cmdFlag.NeedValue {
				err = cmdFlag.ValueReceiver(nil)
				if err != nil {
					return err
				}
			} else {
				valueReceiver = cmdFlag.ValueReceiver
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

func (r *CommandRequest) WriteInt(i int64) error {
	_, err := r.w.WriteString(fmt.Sprintf(":%d\r\n", i))
	return err
}

func (r *CommandRequest) WriteSimpleString(s string) error {
	_, err := r.w.WriteString(fmt.Sprintf("+%s\r\n", s))
	return err
}

func (r *CommandRequest) WriteBulkString(s []byte) error {
	_, err := r.w.WriteString(fmt.Sprintf("$%d\r\n", len(s)))
	if err != nil {
		return err
	}

	_, err = r.w.Write(s)
	if err != nil {
		return err
	}

	_, err = r.w.Write([]byte("\r\n"))
	if err != nil {
		return err
	}

	return err
}

func (r *CommandRequest) WriteNullBulkString() error {
	_, err := r.w.WriteString("$-1\r\n")
	return err
}

func (r *CommandRequest) WriteArrayLen(i int) error {
	_, err := r.w.WriteString(fmt.Sprintf("*%d\r\n", i))
	return err
}

func (r *CommandRequest) WriteNullArray() error {
	_, err := r.w.WriteString("$-1\r\n")
	return err
}

func (r *CommandRequest) WriteError(s string) error {
	err := r.DiscardAllArgs()
	if err != nil {
		return err
	}

	_, err = r.w.WriteString(fmt.Sprintf("-%s\r\n", s))
	return err
}

func (r *CommandRequest) FlushWrites() error {
	return r.w.Flush()
}
