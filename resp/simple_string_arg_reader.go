package resp

type simpleStringArgReader struct {
	args   [][]byte
	argCur int
}

func (r *simpleStringArgReader) ArgCount() int {
	return len(r.args) - r.argCur
}

func (r *simpleStringArgReader) ReadArg(count int) ([][]byte, error) {
	if r.argCur+count > len(r.args) {
		return nil, ErrNoMoreArguments
	}

	ss := r.args[r.argCur : r.argCur+count]
	r.argCur += count
	return ss, nil
}

func (r *simpleStringArgReader) DiscardAllArgs() error {
	r.argCur = len(r.args)
	return nil
}
