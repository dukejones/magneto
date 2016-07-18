package cache

import (
	"io"
	"net/url"
	"log"
	"strings"
)

// TODO: move this to proper location in binary object store adapter
func UrlToRetrievalPath(url url.URL) string {
	return strings.Replace(url.Path[1:], "/", "-", -1)
}

type CacheWriter struct {
	MetadataStore MetadataStore
	BinaryStore BinaryStore
}

func (cw CacheWriter) Save(contentType string, url url.URL, bodyReader io.ReadCloser) {
	retrievalPath := UrlToRetrievalPath(url)
	log.Println("Now Caching", contentType, url.Path, retrievalPath)

	metadata := Metadata{
		ContentType: contentType,
		RetrievalPath: retrievalPath,
	}
	cw.MetadataStore.Set(url.Path, metadata)
	cw.BinaryStore.Set(retrievalPath, bodyReader)
}

func (cw CacheWriter) Exists(url url.URL) (bool, error) {
	exists, err := cw.MetadataStore.Exists(url.Path)
	return exists, err
}

