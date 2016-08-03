package cache

import (
	"io"
	"fmt"
)

// RetrievalPath is the binary retrieval path; this differs based 
// on what BinaryStore is being used.
type Metadata struct {
	ContentType string
	Size int64
}

type MetadataStore interface {
	Get(key string) (*Metadata, error)
	Set(key string, metadata *Metadata) error
}

type BinaryStore interface {
	Get(key string) (io.ReadCloser, error)
	Set(key string, reader io.ReadCloser) error
}

type CacheStore struct {
	MetadataStore MetadataStore
	BinaryStore BinaryStore
}

func (cs CacheStore) Get(key string) (*Metadata, io.ReadCloser, error) {
	fmt.Println(key)
	metadata, err := cs.MetadataStore.Get(key)
	if err != nil {
		return metadata, nil, err
	}
	reader, err := cs.BinaryStore.Get(key)
	return metadata, reader, err
}

func (cs CacheStore) Set(key string, metadata *Metadata, binaryData io.ReadCloser) error {
	if err := cs.BinaryStore.Set(key, binaryData); err != nil {
		return err
	}
	return cs.MetadataStore.Set(key, metadata)
}
