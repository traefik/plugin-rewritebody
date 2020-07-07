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
	Rewrites []Rewrite `json:"rewrites,omitempty"`
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
	name     string
	next     http.Handler
	rewrites []*rewrite
}

type bufferedResponseWriter struct {
	http.ResponseWriter
	buffer bytes.Buffer
}

func (b *bufferedResponseWriter) Write(p []byte) (int, error) {
	return b.buffer.Write(p)
}

// New creates a new handler.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]*rewrite, len(config.Rewrites))
	for i, rewriteConfig := range config.Rewrites {
		filterRegexp, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex expression %q: %w", rewriteConfig.Regex, err)
		}

		rewrites[i] = &rewrite{
			regex:       filterRegexp,
			replacement: []byte(rewriteConfig.Replacement),
		}
	}

	return &rewriteBody{
		name:     name,
		next:     next,
		rewrites: rewrites,
	}, nil
}

func (r *rewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	brw := &bufferedResponseWriter{
		ResponseWriter: rw,
	}

	r.next.ServeHTTP(brw, req)

	bodyBytes := brw.buffer.Bytes()

	if contentEncoding := brw.Header().Get("Content-Encoding"); contentEncoding != "" && contentEncoding != "identity" {
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write rewrited body: %v", err)
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
