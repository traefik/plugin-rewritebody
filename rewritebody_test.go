package plugin_rewritebody

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeHTTP(t *testing.T) {
	tests := []struct {
		desc            string
		contentEncoding string
		rewrites        []Rewrite
		lastModified    bool
		resBody         string
		expResBody      string
		expLastModified bool
	}{
		{
			desc: "should replace foo by bar",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
			},
			resBody:    "foo is the new bar",
			expResBody: "bar is the new bar",
		},
		{
			desc: "should replace foo by bar, then by foo",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
				{
					Regex:       "bar",
					Replacement: "foo",
				},
			},
			resBody:    "foo is the new bar",
			expResBody: "foo is the new foo",
		},
		{
			desc: "should not replace anything if content encoding is not identity or empty",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
			},
			contentEncoding: "gzip",
			resBody:         "foo is the new bar",
			expResBody:      "foo is the new bar",
		},
		{
			desc: "should replace foo by bar if content encoding is identity",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
			},
			contentEncoding: "identity",
			resBody:         "foo is the new bar",
			expResBody:      "bar is the new bar",
		},
		{
			desc: "should not remove the last modified header",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
			},
			contentEncoding: "identity",
			lastModified:    true,
			resBody:         "foo is the new bar",
			expResBody:      "bar is the new bar",
			expLastModified: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				LastModified: test.lastModified,
				Rewrites:     test.rewrites,
			}

			next := func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Set("Content-Encoding", test.contentEncoding)
				rw.Header().Set("Last-Modified", "Thu, 02 Jun 2016 06:01:08 GMT")

				rw.WriteHeader(http.StatusOK)

				_, _ = fmt.Fprintf(rw, test.resBody)
			}

			rewriteBody, err := New(context.Background(), http.HandlerFunc(next), config, "rewriteBody")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			rewriteBody.ServeHTTP(recorder, req)

			if _, exists := recorder.Result().Header["Last-Modified"]; exists != test.expLastModified {
				t.Errorf("got last-modified header %v, want %v", exists, test.expLastModified)
			}

			if !bytes.Equal([]byte(test.expResBody), recorder.Body.Bytes()) {
				t.Errorf("got body %q, want %q", recorder.Body.Bytes(), test.expResBody)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		desc     string
		rewrites []Rewrite
		expErr   bool
	}{
		{
			desc: "should return no error",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
				{
					Regex:       "bar",
					Replacement: "foo",
				},
			},
			expErr: false,
		},
		{
			desc: "should return an error",
			rewrites: []Rewrite{
				{
					Regex:       "*",
					Replacement: "bar",
				},
			},
			expErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				Rewrites: test.rewrites,
			}

			_, err := New(context.Background(), nil, config, "rewriteBody")
			if test.expErr && err == nil {
				t.Fatal("expected error on bad regexp format")
			}
		})
	}
}
