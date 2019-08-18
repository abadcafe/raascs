package resp

import (
	"bufio"
)

type bulkStringArgReader struct {
	r        *bufio.Reader
	argCount int
	argCur   int
}

func (r *bulkStringArgReader) ArgCount() int {
	return r.argCount
}

func (r *bulkStringArgReader) ReadArg(count int) ([]string, error) {
	var ss []string

	for i := 0; i < count; i++ {
		if r.argCount <= 0 {
			return nil, ErrNoMoreArguments
		}

		buf, err := readBulkString(r.r)
		if err != nil {
			return nil, err
		}

		ss = append(ss, string(buf))
		r.argCount--
	}

	return ss, nil
}

func (r *bulkStringArgReader) DiscardAllArgs() error {
	_, err := r.ReadArg(r.ArgCount())
	return err
}
