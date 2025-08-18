package queue

import (
	"bytes"
)

type readerFile struct {
	reader      *bytes.Reader
	size        int64
	contentType string
}

func (rf *readerFile) Read(p []byte) (n int, err error) {
	return rf.reader.Read(p)
}

func (rf *readerFile) ReadAt(p []byte, off int64) (n int, err error) {
	return rf.reader.ReadAt(p, off)
}

func (rf *readerFile) Seek(offset int64, whence int) (int64, error) {
	return rf.reader.Seek(offset, whence)
}

func (rf *readerFile) Close() error {
	return nil
}

func (rf *readerFile) Size() int64 {
	return rf.size
}
