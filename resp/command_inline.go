package resp

import "io"

type inlineCommand struct {
	commandWriter
	args   []string
	argCur int
}

func (c *inlineCommand) ArgCount() int {
	return len(c.args)
}

func (c *inlineCommand) ReadArg() (string, error) {
	if c.argCur >= len(c.args) {
		return "", io.EOF
	}

	arg := c.args[c.argCur]
	c.argCur++
	return arg, nil
}
