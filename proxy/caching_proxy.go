package proxy

import (
	"io"
	"net/url"
	"net/http"
	"net/http/httputil"
	"strconv"
	"log"
	"github.com/dukejones/magneto/cache"
)

const (
	BlockSize = 4096
)

type CachingProxy struct {
	target *url.URL
	delegate *httputil.ReverseProxy
	cache cache.CacheStore
}

func New(target string, cacheStore cache.CacheStore) *CachingProxy {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = NewMultiWriterTransport(cacheStore)
	return &CachingProxy{
		target: url,
		delegate: proxy,
		cache: cacheStore,
	}
}

func (proxy *CachingProxy) Handler(w http.ResponseWriter, req *http.Request) {
	metadata, r, err := proxy.cache.Get(req.URL.Path)
	if err != nil {
		log.Println("Error retrieving from cache:", err)
	}
	tooSmall := (metadata != nil && metadata.Size < BlockSize)
	log.Println("tooSmall", tooSmall)
	if metadata != nil {
		log.Println(strconv.FormatInt(metadata.Size, 10))
	}
	if r == nil || metadata == nil || tooSmall {
		if !tooSmall {
			log.Println(req.URL.Path, "not found in cache.")
		}
		proxy.delegate.ServeHTTP(w, req)
	} else {
		w.Header().Set("Content-Type", metadata.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(metadata.Size, 10))
		n, err := io.CopyBuffer(w, r, nil)
		if err != nil {
			log.Println("Unable to serve image:", err)
		} else if n > 0 {
			log.Println("Served", n, "bytes from cache.")
		}
	}
}

