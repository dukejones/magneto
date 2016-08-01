package cache

import (
	"gopkg.in/redis.v4"
	"testing"
)
const (
	largeSize = 4234987444
)

func TestSet(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 13}) 
	subject := RedisMetadataStore{redisClient}
	if err := subject.Set("key", &Metadata{"text/html", largeSize}); err != nil {
		t.Error("Cannot set Redis key")
	}
	metadata, err := subject.Get("key")
	if  err != nil {
		t.Fatal("Cannot retrieve Redis key")
	}
	t.Log("Metadata retrieved:", metadata, err)
	if metadata.Size != largeSize || metadata.ContentType != "text/html" {
		t.Error("Wrong Metadata retrieved:", metadata)
	}
	redisClient.FlushDb()
}