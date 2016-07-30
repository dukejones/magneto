package cache

import (
	"os"
	"io"
	"strings"
)
type FileStore struct {
	Directory string
}

func NewFileStore(dir string) *FileStore {
	os.Mkdir(dir, os.ModePerm)
	return &FileStore{
		Directory: dir,
	}
}

func (fs *FileStore) GetFile(key string) (*os.File, error) {
	retrievalPath := keyToRetrievalPath(key)
	return os.Open(fs.Directory + "/" + retrievalPath)
}

func (fs *FileStore) Get(key string) (io.ReadCloser, error) {
	return fs.GetFile(key)
}

func (fs *FileStore) Set(key string, reader io.ReadCloser) error {
	retrievalPath := keyToRetrievalPath(key)
	file, err := os.Create(fs.Directory + "/" + retrievalPath)
	defer file.Close()
	defer reader.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(file, reader)
	return err
}

func (fs *FileStore) GetSize(key string) (int64, error) {
	file, err := fs.GetFile(key)
	fileInfo, err := file.Stat()
	return fileInfo.Size(), err
}

func keyToRetrievalPath(key string) string {
	return strings.Replace(key, "/", "-", -1)
}
