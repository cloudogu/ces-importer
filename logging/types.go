package logging

import (
	"io"
	"os"
)

type file interface {
	io.Writer
	WriteString(s string) (n int, err error)
	Close() error
}

type osOpenFile func(name string, flag int, perm os.FileMode) (file, error)
type ioCopy func(dst io.Writer, src io.Reader) (written int64, err error)
type createMultiWriter func(writers ...io.Writer) io.Writer
