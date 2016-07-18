package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"net/http"
	"gopkg.in/redis.v4"
	"github.com/dukejones/magneto/proxy"
	"github.com/dukejones/magneto/cache"
)
// TODO: https://github.com/oxtoacart/bpool

const(
	tmpDir = "/tmp/magneto-files"
)


///////////////////////////////////////////////////


type RedisStore struct {
	client *redis.Client
}

func (rs *RedisStore) Get(key string) (cache.Metadata, error) {
	val, err := rs.client.HMGet(key, "ContentType", "RetrievalPath").Result()
	return cache.Metadata{
		ContentType: val[0].(string),
		RetrievalPath: val[1].(string),
	}, err
}

func (rs *RedisStore) Set(key string, metadata cache.Metadata) error {
	metaMap := map[string]string{
		"ContentType": metadata.ContentType,
		"RetrievalPath": metadata.RetrievalPath,
	}
	return rs.client.HMSet(key, metaMap).Err()
}

func (rs *RedisStore) Exists(key string) (bool, error) {
	exists, err := rs.client.Exists(key).Result()
	return exists, err
}

///////////////////////////////////////////////////////////////////////////

type FileStore struct {
	Directory string
}

func NewFileStore() *FileStore {
	os.Mkdir(tmpDir, os.ModePerm)
	return &FileStore{
		Directory: tmpDir,
	}
}

func (fs *FileStore) Get(retrievalPath string) (io.ReadCloser, int64, error) {
	file, err := os.Open(fs.Directory + "/" + retrievalPath)
	if err != nil {
		log.Println(err)
	}
	fileInfo, _ := file.Stat()
	
	return file, fileInfo.Size(), err
}

func (fs *FileStore) Set(RetrievalPath string, r io.ReadCloser) error {
	file, err := os.Create(fs.Directory + "/" + RetrievalPath)
	defer file.Close()
	defer r.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	if _, err := io.Copy(file, r); err != nil {
		log.Println("Saving To File:", err)
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////


func main() {
	fileStore := NewFileStore()

	redisStore := &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0, // use default DB
		}),
	}

	fmt.Println("Listening on port 8888...")
	p := proxy.New("http://localhost:8081", cache.CacheWriter{redisStore, fileStore})

	http.ListenAndServe(":8888", http.HandlerFunc(p.Handler))
}

