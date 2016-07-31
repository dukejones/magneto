package cache

import "testing"

func TestSet(t *testing.T) {
	object := RedisMetadataStore{redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 13}) }
	err := object.Set("key", Metadata{"text/html", 1024})

}