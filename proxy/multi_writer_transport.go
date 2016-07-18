package proxy

import (
	"io"
	"net/http"
	"github.com/dukejones/magneto/cache"
)

// MultiWriterTransport wraps a transport and writes to a separate writer.
type MultiWriterTransport struct {
	TransportDelegate http.RoundTripper
	CacheWriter cache.CacheWriter
}

func NewMultiWriterTransport(cw cache.CacheWriter) *MultiWriterTransport {
	return &MultiWriterTransport{
		TransportDelegate: http.DefaultTransport,
		CacheWriter: cw,
	}
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

func (mwt MultiWriterTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, error := mwt.TransportDelegate.RoundTrip(request)

	cacheBodyReader, cacheBodyWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	multiWriter := io.MultiWriter(responseWriter, cacheBodyWriter)

	go func() {
		io.CopyBuffer(multiWriter, response.Body, nil)
		response.Body.Close()
		responseWriter.Close()
		cacheBodyWriter.Close()
	}()

	go func() {
		mwt.CacheWriter.Save(response.Header.Get("Content-Type"), *response.Request.URL, cacheBodyReader)
	}()

	newResponse := copyResponse(response, responseReader)
	return newResponse, error
}
