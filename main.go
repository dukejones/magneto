package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"gopkg.in/redis.v4"
)
// TODO: https://github.com/oxtoacart/bpool

const(
	tmpDir = "/tmp/magneto-files"
)

type Metadata struct {
	ContentType string
	RetrievalPath string
}

type MetadataStore interface {
	Get(key string) (Metadata, error) 
	Set(key string, metadata Metadata) error
}

type RedisStore struct {
	client *redis.Client
}

func (rs *RedisStore) Get(key string) (Metadata, error) {
	val, err := rs.client.HMGet(key, "ContentType", "RetrievalPath").Result()
	return Metadata{
		ContentType: val[0].(string),
		RetrievalPath: val[1].(string),
	}, err
}

func (rs *RedisStore) Set(key string, metadata Metadata) error {
	metaMap := map[string]string{
		"ContentType": metadata.ContentType,
		"RetrievalPath": metadata.RetrievalPath,
	}
	return rs.client.HMSet(key, metaMap).Err()
}
///////////////////////////////////////////////////////////////////////////

type BinaryStore interface {
	Get(retrievalPath string) (io.Reader, error)
	Set(retrievalPath string, r io.Reader) error
}

type FileStore struct {
	Directory string
}

func NewFileStore() *FileStore {
	os.Mkdir(tmpDir, os.ModePerm)
	return &FileStore{
		Directory: tmpDir,
	}
}

func (fs *FileStore) Get(retrievalPath string) (io.Reader, error) {
	file, err := os.Open(fs.Directory + "/" + retrievalPath)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return file, nil
}

func (fs *FileStore) Set(retrievalPath string, r io.Reader) error {
	file, err := os.Create(fs.Directory + "/" + retrievalPath)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
		return err
	}

	// buf := httputil.BufferPool.Get()
	// io.CopyBuffer(file, reader, buf)
	// httputil.BufferPool.Put(buf)
	if _, err := io.Copy(file, r); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////

type CacheWriter struct {
	mdStore MetadataStore
	bStore BinaryStore
}

// MultiWriterTransport wraps a transport and writes to a separate writer.
type MultiWriterTransport struct {
	TransportDelegate http.RoundTripper
	Writer CacheWriter
}

func NewMultiWriterTransport(Writer CacheWriter) *MultiWriterTransport {
	return &MultiWriterTransport{
		TransportDelegate: http.DefaultTransport,
		Writer: Writer,
	}
}

func (mwt MultiWriterTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, error := mwt.TransportDelegate.RoundTrip(request)
	// magic happens here
	// originalReader := response.Body // io.ReadCloser

	return response, error
}

///////////////////////////////////////////////////////////////////////////
type CachingProxy struct {
	target *url.URL
	proxy *httputil.ReverseProxy
	cache CacheWriter
}

func NewCachingProxy(target string, cacheWriter CacheWriter) *CachingProxy {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = MultiWriterTransport{}

	return &CachingProxy{
		target: url,
		proxy: proxy,
		cache: cacheWriter,
	}
}

func (p *CachingProxy) handle(w http.ResponseWriter, r *http.Request) {
	// if p.cache.Get("key")
	p.proxy.ServeHTTP(w, r)
}

/////////////////////////////////////////////////////
	/*
Serve an image:
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		if _, err := w.Write(bytes); err != nil {
			log.Println("unable to write image.")
		}
	}
	*/


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
	p := NewCachingProxy("http://localhost:8081", CacheWriter{redisStore, fileStore})

	http.ListenAndServe(":8888", p.proxy)
}

