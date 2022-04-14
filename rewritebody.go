// Package plugin_rewritebody a plugin to rewrite response body.
package plugin_rewritebody

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
)

// var (
// 	// LoggerINFO Main logger
// 	LoggerINFO = log.New(ioutil.Discard, "INFO: Fail2Ban: ", log.Ldate|log.Ltime|log.Lshortfile)
// 	// LoggerDEBUG debug logger
// 	LoggerDEBUG = log.New(ioutil.Discard, "DEBUG: Fail2Ban: ", log.Ldate|log.Ltime|log.Lshortfile)
// )

// Rewrite holds one rewrite body configuration.
type Rewrite struct {
	Regex       string `json:"regex,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// Config holds the plugin configuration.
type Config struct {
	LastModified bool      `json:"lastModified,omitempty"`
	Rewrites     []Rewrite `json:"rewrites,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type rewrite struct {
	regex       *regexp.Regexp
	replacement []byte
}

type rewriteBody struct {
	name         string
	next         http.Handler
	rewrites     []rewrite
	lastModified bool
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]rewrite, len(config.Rewrites))

	for index, rewriteConfig := range config.Rewrites {
		regex, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
		}

		rewrites[index] = rewrite{
			regex:       regex,
			replacement: []byte(rewriteConfig.Replacement),
		}
	}

	return &rewriteBody{
		name:         name,
		next:         next,
		rewrites:     rewrites,
		lastModified: config.LastModified,
	}, nil
}

func (bodyRewrite *rewriteBody) ServeHTTP(response http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		lastModified:   bodyRewrite.lastModified,
		ResponseWriter: response,
	}

	bodyRewrite.next.ServeHTTP(wrappedWriter, req)

	// bodyBytes := wrappedWriter.buffer.Bytes()

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")
	contentType := strings.ToLower(wrappedWriter.Header().Get("Content-Type"))
	isText := contentType == "" || strings.Contains(contentType, "text")

	isGzip := contentEncoding == "gzip"

	if !isText || (!isGzip && contentEncoding != "" && contentEncoding != "identity") {
		if _, err := response.Write(wrappedWriter.buffer.Bytes()); err != nil {
			log.Printf("unable to write body: %v", err)
		}

		return
	}

	bodyBytes := wrappedWriter.decompressBody(isGzip)

	for _, rwt := range bodyRewrite.rewrites {
		bodyBytes = rwt.regex.ReplaceAll(bodyBytes, rwt.replacement)
	}

	preparedBody := prepareBodyBytes(bodyBytes, isGzip)

	if _, err := response.Write(preparedBody); err != nil {
		log.Printf("unable to write rewrited body: %v", err)
	}
}

func (wrappedWriter *responseWriter) decompressBody(isGzip bool) (bodyBytes []byte) {
	if !isGzip {
		return wrappedWriter.buffer.Bytes()
	}

	zReader, err := gzip.NewReader(&wrappedWriter.buffer)
	if err != nil {
		log.Printf("Failed to load body reader: %v", err)

		return wrappedWriter.buffer.Bytes()
	}

	bodyBytes, err = io.ReadAll(zReader)

	if err != nil {
		log.Printf("Failed to read body: %v", err)

		return wrappedWriter.buffer.Bytes()
	}

	err = zReader.Close()

	if err != nil {
		log.Printf("Failed to close reader: %v", err)

		return wrappedWriter.buffer.Bytes()
	}

	return bodyBytes
}

func prepareBodyBytes(bodyBytes []byte, isGzip bool) (b []byte) {
	if !isGzip {
		return bodyBytes
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(bodyBytes); err != nil {
		log.Printf("unable to recompress rewrited body: %v", err)

		return bodyBytes
	}

	if err := gzipWriter.Close(); err != nil {
		log.Printf("unable to close gzip writer: %v", err)

		return bodyBytes
	}

	return buf.Bytes()
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool

	http.ResponseWriter
}

func (wrappedWriter *responseWriter) WriteHeader(statusCode int) {
	if !wrappedWriter.lastModified {
		wrappedWriter.ResponseWriter.Header().Del("Last-Modified")
	}

	wrappedWriter.wroteHeader = true

	// Delegates the Content-Length Header creation to the final body write.
	wrappedWriter.ResponseWriter.Header().Del("Content-Length")

	wrappedWriter.ResponseWriter.WriteHeader(statusCode)
}

func (wrappedWriter *responseWriter) Write(p []byte) (int, error) {
	if !wrappedWriter.wroteHeader {
		wrappedWriter.WriteHeader(http.StatusOK)
	}

	return wrappedWriter.buffer.Write(p)
}

func (wrappedWriter *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := wrappedWriter.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", wrappedWriter.ResponseWriter)
	}

	return hijacker.Hijack()
}

func (wrappedWriter *responseWriter) Flush() {
	if flusher, ok := wrappedWriter.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
