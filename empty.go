package docker

import "io"

type emptyReader struct{}

func (r *emptyReader) Read(b []byte) (int, error) {
	return 0, io.EOF
}
func (r *emptyReader) Close() error {
	return nil
}

var empty = &emptyReader{}
