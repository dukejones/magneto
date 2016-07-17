package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"strconv"
	"strings"
	"net/http"
	"net/http/httputil"
	"net/url"
	"gopkg.in/redis.v4"
)
// TODO: https://github.com/oxtoacart/bpool

const(
	tmpDir = "/tmp/magneto-files"
)

func UrlToRetrievalPath(url url.URL) string {
	return strings.Replace(url.Path[1:], "/", "-", -1)
}

///////////////////////////////////////////////////

type Metadata struct {
	ContentType string
	RetrievalPath string
}

type MetadataStore interface {
	Get(key string) (Metadata, error) 
	Set(key string, metadata Metadata) error
	Exists(key string) (bool, error)
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

func (rs *RedisStore) Exists(key string) (bool, error) {
	exists, err := rs.client.Exists(key).Result()
	return exists, err
}

///////////////////////////////////////////////////////////////////////////

type BinaryStore interface {
	Get(retrievalPath string) (io.ReadCloser, int64, error)
	Set(retrievalPath string, r io.ReadCloser) error
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

type CacheWriter struct {
	mdStore MetadataStore
	bStore BinaryStore
}

func (cw CacheWriter) Save(contentType string, url url.URL, bodyReader io.ReadCloser) {
	retrievalPath := UrlToRetrievalPath(url)
	log.Println("Now Caching", contentType, url.Path, retrievalPath)

	metadata := Metadata{
		ContentType: contentType,
		RetrievalPath: retrievalPath,
	}
	cw.mdStore.Set(url.Path, metadata)
	cw.bStore.Set(retrievalPath, bodyReader)
}

func (cw CacheWriter) Exists(url url.URL) (bool, error) {
	exists, err := cw.mdStore.Exists(url.Path)
	return exists, err
}

// MultiWriterTransport wraps a transport and writes to a separate writer.
type MultiWriterTransport struct {
	TransportDelegate http.RoundTripper
	CacheWriter CacheWriter
}

func NewMultiWriterTransport(cw CacheWriter) *MultiWriterTransport {
	return &MultiWriterTransport{
		TransportDelegate: http.DefaultTransport,
		CacheWriter: cw,
	}
}

func copyResponse(src *http.Response, body io.ReadCloser) *http.Response {
	headerPtr := &src.Header
	transferEncodingPtr := &src.TransferEncoding
	trailerPtr := &src.Trailer

	return &http.Response{
		Status: src.Status,
		StatusCode: src.StatusCode,
		Proto: src.Proto,
		ProtoMajor: src.ProtoMajor,
		ProtoMinor: src.ProtoMinor,
		Header: *headerPtr,
		Body: body,
		ContentLength: src.ContentLength,
		TransferEncoding: *transferEncodingPtr,
		Close: false,
		Trailer: *trailerPtr,
		Request: src.Request,
		TLS: src.TLS,
	}
}

func (mwt MultiWriterTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, error := mwt.TransportDelegate.RoundTrip(request)

	cacheBodyReader, cacheBodyWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	multiWriter := io.MultiWriter(responseWriter, cacheBodyWriter)

	go func() {
		io.CopyBuffer(multiWriter, response.Body, nil)
		response.Body.Close()
		responseWriter.Close()
		cacheBodyWriter.Close()
	}()

	go func() {
		mwt.CacheWriter.Save(response.Header.Get("Content-Type"), *response.Request.URL, cacheBodyReader)
	}()

	newResponse := copyResponse(response, responseReader)
	return newResponse, error
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
	proxy.Transport = NewMultiWriterTransport(cacheWriter)
	return &CachingProxy{
		target: url,
		proxy: proxy,
		cache: cacheWriter,
	}
}

func (p *CachingProxy) handle(w http.ResponseWriter, r *http.Request) {
	exists, _ := p.cache.Exists(*r.URL)
	if exists {
		path := r.URL.Path
		metadata, _ := p.cache.mdStore.Get(path)

		file, size, _ := p.cache.bStore.Get(metadata.RetrievalPath)

		w.Header().Set("Content-Type", metadata.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		n, err := io.CopyBuffer(w, file, nil)
		if err != nil {
			log.Println("Unable to write image.", err)
		} else if n > 0 {
			log.Println("Served", n, "bytes from cache.")
		}
	} else {
		p.proxy.ServeHTTP(w, r)
	}
}

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
	p := NewCachingProxy("http://localhost:8081", CacheWriter{redisStore, fileStore})

	http.ListenAndServe(":8888", http.HandlerFunc(p.handle))
}

