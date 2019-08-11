package resp

import "bufio"

type normalCommand struct {
	commandWriter
	r        *bufio.Reader
	argCount int
}

func (c *normalCommand) ArgCount() int {
	return c.argCount
}

func (c *normalCommand) ReadArg() (string, error) {
	buf, err := readBulkString(c.r)
	if err != nil {
		return "", nil
	}

	return string(buf), nil
}
