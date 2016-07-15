package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"gopkg.in/gographics/imagick.v2/imagick"
	"gopkg.in/redis.v4"
)


/////////////////////////////////////////////////
// ImageMagick Resize
func resize(input string, output string) {
	imagick.Initialize()
	defer imagick.Terminate()
	var err error

	mw := imagick.NewMagickWand()

	err = mw.ReadImage(input)
	if err != nil {
		panic(err)
	}

	// Get original logo size
	width := mw.GetImageWidth()
	height := mw.GetImageHeight()

	// Calculate half the size
	hWidth := uint(width / 2)
	hHeight := uint(height / 2)

	// Resize the image using the Lanczos filter
	// The blur factor is a float, where > 1 is blurry, < 1 is sharp
	err = mw.ResizeImage(hWidth, hHeight, imagick.FILTER_LANCZOS, 1)
	if err != nil {
		panic(err)
	}

	// Set the compression quality to 95 (high quality = low compression)
	err = mw.SetImageCompressionQuality(95)
	if err != nil {
		panic(err)
	}

	err = mw.WriteImage(output)
	if err != nil {
		panic(err)
	}
}

///////////////////////////////////////////////////
type FileStore interface {
	Get(string) io.Reader, error
	Set(string, io.Reader) error
}

/*
type RedisFileStore struct {
	client *redis.Client
}

func NewRedisFileStore() *RedisFileStore {
	client := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0, // use default DB
	})
	return &RedisFileStore{client: client}
}

func WriteFileToRedis(filename string, client *redis.Client) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = client.Set("key", bytes, 0).Err()
	if err != nil {
		return err
	}

	return nil
}


func (rfs *RedisFileStore) Get(key string) []byte {
	readString, err := rfs.client.Get(key).Result()
	if err == redis.Nil {
		return nil
	} else if err != nil {
		panic(err)
	}

	return []byte(readString)
}
*/
///////////////////////////////////////////////////////////////////////////

type CachingProxy struct {
	target *url.URL
	proxy *httputil.ReverseProxy
	cache *RedisFileStore
}

func NewCachingProxy(target string) *CachingProxy {
	url, _ := url.Parse(target)
	return &CachingProxy{
		target: url,
		proxy: httputil.NewSingleHostReverseProxy(url),
		cache: NewRedisFileStore(),
	}
}

func (p *CachingProxy) handle(w http.ResponseWriter, r *http.Request) {
	if p.cache.Get("key")
	p.proxy.ServeHTTP(w, r)
}

/////////////////////////////////////////////////////
func main() {

	// WriteFileToRedis("art-Totemical-1016702.jpeg", client)

	rfs := magneto.NewRedisFileStore()
	// bytes := rfs.ReadFile()

	// fmt.Println("key read")
	// err = ioutil.WriteFile("output.jpg", []byte(readString), 0666)
	// if err != nil {
	// 	panic(err)
	// }


	/*
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		if _, err := w.Write(bytes); err != nil {
			log.Println("unable to write image.")
		}
	}
	*/

	fmt.Println("Listening on port 8888...")

	// http.HandleFunc("/", handler)
	// http.ListenAndServe(":8888", nil)

	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   "localhost:8081",
	})
	http.ListenAndServe(":8888", proxy)
}

