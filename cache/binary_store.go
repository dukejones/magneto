package cache

import (
	"io"

)

type BinaryStore interface {
	Get(retrievalPath string) (io.ReadCloser, int64, error)
	Set(retrievalPath string, r io.ReadCloser) error
}

