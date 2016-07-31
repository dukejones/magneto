package cache

import (
	"gopkg.in/redis.v4"
)

// TODO: this implements MetadataStore, rename it to reflect this.
type RedisMetadataStore struct {
	Client *redis.Client
}

func (rms *RedisMetadataStore) Get(key string) (*Metadata, error) {
	val, err := rms.Client.HMGet(key, "ContentType", "Size").Result()
	if err != nil {
		return nil, err
	}

	contentType, ok1 := val[0].(string)
	size, ok2 := val[1].(int64)
	if !ok1 || !ok2 {
		return nil, nil
	}
	return &Metadata{contentType, size}, nil
}

func (rms *RedisMetadataStore) Set(key string, metadata *Metadata) error {
	metaMap := metadata.toMap()
	return rms.Client.HMSet(key, *metaMap).Err()
}
