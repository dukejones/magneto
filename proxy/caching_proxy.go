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

type CachingProxy struct {
	target *url.URL
	proxy *httputil.ReverseProxy
	cache cache.CacheWriter
}

func New(target string, cacheWriter cache.CacheWriter) *CachingProxy {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = NewMultiWriterTransport(cacheWriter)
	return &CachingProxy{
		target: url,
		proxy: proxy,
		cache: cacheWriter,
	}
}

func (p *CachingProxy) Handler(w http.ResponseWriter, r *http.Request) {
	exists, _ := p.cache.Exists(*r.URL)
	if exists {
		path := r.URL.Path
		metadata, _ := p.cache.MetadataStore.Get(path)

		file, size, _ := p.cache.BinaryStore.Get(metadata.RetrievalPath)

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

