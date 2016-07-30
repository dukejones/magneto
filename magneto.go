package main

import (
	"fmt"
	"net/http"
	"gopkg.in/redis.v4"
	"github.com/dukejones/magneto/proxy"
	"github.com/dukejones/magneto/cache"
)

// TODO: https://github.com/oxtoacart/bpool

const(
	tmpDir = "/tmp/magneto-files"
)

func main() {
	fileStore := cache.NewFileStore(tmpDir)

	redisStore := &cache.RedisMetadataStore{
		Client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0, // use default DB
		}),
	}

	fmt.Println("Listening on port 8888...")
	p := proxy.New("http://localhost:8081", cache.CacheStore{redisStore, fileStore})

	http.ListenAndServe(":8888", http.HandlerFunc(p.Handler))
}
