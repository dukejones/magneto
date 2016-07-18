package cache

type Metadata struct {
	ContentType string
	RetrievalPath string
}

type MetadataStore interface {
	Get(key string) (Metadata, error) 
	Set(key string, metadata Metadata) error
	Exists(key string) (bool, error)
}
