package uhttp

import (
	"net/http"
	"time"
)

// ResponseWriter is a wrapper around http.ResponseWriter that provides additional information.
// It is used to capture the status code and the number of bytes written. It also provides a way to determine if the
// header has been written.
type ResponseWriter struct {
	http.ResponseWriter
	statusCode      int
	bytesWritten    uint64
	isStatusWritten bool
	startTime       time.Time

	defaultStatusCode     int
	defaultHeaders        map[string]string
	defaultHeadersWritten bool
}

// NewResponseWriter creates a new ResponseWriter.
func NewResponseWriter(w http.ResponseWriter, opts ...WriterOpt) *ResponseWriter {
	rw := &ResponseWriter{
		ResponseWriter: w,
		startTime:      time.Now().UTC(),
		defaultHeaders: map[string]string{
			HeaderContentType: ContentTypeJSON,
		},
		defaultStatusCode: http.StatusOK,
	}

	for _, opt := range opts {
		opt(rw)
	}

	return rw
}

// Write writes the data to the connection as part of an HTTP reply.
func (c *ResponseWriter) Write(p []byte) (bytes int, err error) {
	c.writeDefaultHeaders()
	c.WriteHeader(c.defaultStatusCode)
	bytes, err = c.ResponseWriter.Write(p)
	if err != nil {
		return
	}
	writtenBytes := uint64(0)
	if bytes > 0 {
		writtenBytes = uint64(bytes)
	}
	c.bytesWritten += writtenBytes
	return
}

// WriteHeader sends an HTTP response header with the provided status code.
func (c *ResponseWriter) WriteHeader(code int) {
	if c.isStatusWritten {
		return
	}
	c.writeDefaultHeaders()
	c.ResponseWriter.WriteHeader(code)
	c.isStatusWritten = true
	c.statusCode = code
}

// StatusCode returns the status code.
func (c *ResponseWriter) StatusCode() int {
	if !c.isStatusWritten || c.statusCode == 0 {
		return c.defaultStatusCode
	}
	return c.statusCode
}

// BytesWritten returns the number of bytes written.
func (c *ResponseWriter) BytesWritten() uint64 {
	return c.bytesWritten
}

// IsHeaderWritten returns true if the header has been written.
func (c *ResponseWriter) IsHeaderWritten() bool {
	return c.isStatusWritten
}

// GetRequestDuration gets the duration of the request
func (c *ResponseWriter) GetRequestDuration() time.Duration {
	return time.Since(c.startTime)
}

func (c *ResponseWriter) writeDefaultHeaders() {
	if c.defaultHeadersWritten {
		return
	}
	c.defaultHeadersWritten = true

	if c.defaultHeaders == nil {
		return
	}

	for header, value := range c.defaultHeaders {
		if c.Header().Get(header) != "" {
			// If the header has already been set, do not overwrite it.
			continue
		}
		c.Header().Set(header, value)
	}
}
