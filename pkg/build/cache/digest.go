package cache

import (
	"fmt"
	"hash"
	"os"
)

type DigestWriter struct {
	f   *os.File
	md5 hash.Hash
}

func newDigestWriter(f *os.File, md5 hash.Hash) *DigestWriter {
	return &DigestWriter{
		f:   f,
		md5: md5,
	}
}

func (dw *DigestWriter) Write(p []byte) (n int, err error) {
	n1, err1 := dw.f.Write(p)
	n2, err2 := dw.md5.Write(p)

	if n1 != n2 {
		return 0, fmt.Errorf("digest writer failed: %v, %v", err1, err2)
	}

	if err1 != nil {
		return 0, err1
	}

	if err2 != nil {
		return 0, err2
	}

	return n1, nil
}
