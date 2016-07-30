package proxy

import (
	"io"
	"net/http"
	"github.com/dukejones/magneto/cache"
)

// MultiWriterTransport wraps a transport and writes to a separate writer.
type MultiWriterTransport struct {
	TransportDelegate http.RoundTripper
	CacheStore cache.CacheStore
}

func NewMultiWriterTransport(cs cache.CacheStore) *MultiWriterTransport {
	return &MultiWriterTransport{
		TransportDelegate: http.DefaultTransport,
		CacheStore: cs,
	}
}

func (mwt MultiWriterTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, error := mwt.TransportDelegate.RoundTrip(request)

	cacheBodyReader, cacheBodyWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	multiWriter := io.MultiWriter(responseWriter, cacheBodyWriter)

	go func() {
		defer response.Body.Close()
		defer responseWriter.Close()
		defer cacheBodyWriter.Close()
		io.CopyBuffer(multiWriter, response.Body, nil)
	}()

	go func() {
		url := *response.Request.URL
		metadata := cache.Metadata{
			ContentType: response.Header.Get("Content-Type"),
			Size: response.ContentLength, //response.Header.Get("Content-Length"),
		}
		mwt.CacheStore.Set(url.Path, &metadata, cacheBodyReader)
	}()

	newResponse := copyResponse(response, responseReader)
	return newResponse, error
}

func copyResponse(src *http.Response, body io.ReadCloser) *http.Response {
	headerPtr := &src.Header
	transferEncodingPtr := &src.TransferEncoding
	trailerPtr := &src.Trailer

	return &http.Response{
		Status: src.Status,
		StatusCode: src.StatusCode,
		Proto: src.Proto,
		ProtoMajor: src.ProtoMajor,
		ProtoMinor: src.ProtoMinor,
		Header: *headerPtr,
		Body: body,
		ContentLength: src.ContentLength,
		TransferEncoding: *transferEncodingPtr,
		Close: false,
		Trailer: *trailerPtr,
		Request: src.Request,
		TLS: src.TLS,
	}
}
