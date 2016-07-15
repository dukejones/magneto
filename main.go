package main

import (
	"fmt"
	"log"
	// "os"
	// "io/ioutil"
	// "io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"gopkg.in/redis.v4"
)


type Metadata struct {
	ContentType string
	FilePath string
}

type MetadataStore interface {
	Get(key string) (Metadata, error) 
	Set(key string, metadata Metadata) error
}

type RedisStore struct {
	client *redis.Client
}

func (rs *RedisStore) Get(key string) (Metadata, error) {
	val, err := rs.client.HMGet(key, "ContentType", "FilePath").Result()
	return Metadata{
		ContentType: val[0].(string),
		FilePath: val[1].(string),
	}, err
}

func (rs *RedisStore) Set(key string, metadata Metadata) error {
	metaMap := map[string]string{
		"ContentType": metadata.ContentType,
		"FilePath": metadata.FilePath,
	}
	return rs.client.HMSet(key, metaMap).Err()
}


///////////////////////////////////////////////////////////////////////////

type CachingProxy struct {
	target *url.URL
	proxy *httputil.ReverseProxy
	cache MetadataStore
}

func NewCachingProxy(target string, cache MetadataStore) *CachingProxy {
	url, _ := url.Parse(target)
	return &CachingProxy{
		target: url,
		proxy: httputil.NewSingleHostReverseProxy(url),
		cache: cache,
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

	redisStore := &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0, // use default DB
		}),
	}


	fmt.Println("Listening on port 8888...")

	// http.HandleFunc("/", handler)
	// http.ListenAndServe(":8888", nil)

	p := NewCachingProxy("http://localhost:8081", redisStore)
	log.Fatal(http.ListenAndServe(":8888", p.proxy))
}

