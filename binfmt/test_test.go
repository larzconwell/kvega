package binfmt

import (
	"bufio"
	"bytes"
	"fmt"
)

type errReadWriter struct {
	buf   *bytes.Buffer
	n     int
	errOn int
	err   error
}

func (erw *errReadWriter) Read(p []byte) (int, error) {
	if erw.buf == nil {
		erw.buf = bytes.NewBuffer(nil)
	}

	maxRead := min(len(p), erw.errOn-erw.n)

	n, err := erw.buf.Read(p[:maxRead])
	if err != nil {
		return n, fmt.Errorf("binfmt: failed to read: %w", err)
	}

	erw.n += n

	if erw.n >= erw.errOn {
		return n, erw.err
	}

	return n, nil
}

func (erw *errReadWriter) Write(p []byte) (int, error) {
	if erw.buf == nil {
		erw.buf = bytes.NewBuffer(nil)
	}

	maxWrite := min(len(p), erw.errOn-erw.n)

	n, err := erw.buf.Write(p[:maxWrite])
	if err != nil {
		return n, fmt.Errorf("binfmt: failed to write: %w", err)
	}

	erw.n += n

	if erw.n >= erw.errOn {
		return n, erw.err
	}

	return n, nil
}

func encoderToBufReader(enc *Encoder) (*bufio.Reader, error) {
	var buf bytes.Buffer

	_, err := enc.WriteTo(&buf)
	if err != nil {
		return nil, err
	}

	return bufio.NewReader(&buf), nil
}
