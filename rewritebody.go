package plugin_rewritebody

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

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

type responseWriterWrapper struct {
	buffer       bytes.Buffer
	lastModified bool

	http.ResponseWriter
}

func (r *responseWriterWrapper) WriteHeader(statusCode int) {
	if !r.lastModified {
		r.ResponseWriter.Header().Del("Last-Modified")
	}

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseWriterWrapper) Write(p []byte) (int, error) {
	return r.buffer.Write(p)
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]rewrite, len(config.Rewrites))

	for i, rewriteConfig := range config.Rewrites {
		regex, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
		}

		rewrites[i] = rewrite{
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

func (r *rewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rww := &responseWriterWrapper{
		lastModified:   r.lastModified,
		ResponseWriter: rw,
	}

	r.next.ServeHTTP(rww, req)

	bodyBytes := rww.buffer.Bytes()

	if contentEncoding := rww.Header().Get("Content-Encoding"); contentEncoding != "" && contentEncoding != "identity" {
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write body: %v", err)
		}

		return
	}

	for _, rewrite := range r.rewrites {
		bodyBytes = rewrite.regex.ReplaceAll(bodyBytes, rewrite.replacement)
	}

	if _, err := rw.Write(bodyBytes); err != nil {
		log.Printf("unable to write rewrited body: %v", err)
	}
}
