package cache

import (
	"gopkg.in/redis.v4"
	"strconv"
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

	contentType := val[0].(string)
	size, _ := strconv.ParseUint(val[1].(string), 10, 32)
	return &Metadata{contentType, uint(size)}, nil
}

func (rms *RedisMetadataStore) Set(key string, metadata *Metadata) error {
	metaMap := map[string]string{
		"ContentType": metadata.ContentType,
		"Size": strconv.FormatUint(uint64(metadata.Size), 10),
	}
	return rms.Client.HMSet(key, metaMap).Err()
}
