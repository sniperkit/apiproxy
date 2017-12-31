package apiproxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sniperkit/httpcache"
	// "github.com/sourcegraph/httpcache"
)

// RevalidationTransport is an implementation of net/http.RoundTripper that
// permits custom behavior with respect to cache entry revalidation for
// resources on the target server.
//
// If the request contains cache validators (an If-None-Match or
// If-Modified-Since header), then Check.Valid is called to determine whether
// the cache entry should be revalidated (by being passed to the underlying
// transport). In this way, the Check Validator can effectively extend or
// shorten cache age limits.
//
// If the request does not contain cache validators, then it is passed to the
// underlying transport.
type RevalidationTransport struct {
	// Check.Valid is called on each request in RoundTrip. If it returns true,
	// RoundTrip synthesizes and returns an HTTP 304 Not Modified response.
	// Otherwise, the request is passed through to the underlying transport.
	Check Validator

	// Transport is the underlying transport. If nil, net/http.DefaultTransport is used.
	Transport http.RoundTripper
}

// RoundTrip takes a Request and returns a Response.
func (t *RevalidationTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if t.Check != nil && hasCacheValidator(req.Header) {
		agestr := req.Header.Get(httpcache.XCacheAge)
		if agestr != "" {
			var age time.Duration
			age, err = time.ParseDuration(agestr + "s")
			if err == nil && t.Check.Valid(req.URL, age) {
				resp = &http.Response{
					Request:          req,
					TransferEncoding: req.TransferEncoding,
					StatusCode:       http.StatusNotModified,
					Body:             ioutil.NopCloser(bytes.NewReader([]byte(""))),
				}
				return
			}
		}
	}

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	return transport.RoundTrip(req)
}

// hasCacheValidator returns true if the headers contain cache validators. See
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html#sec13.3 for more
// information.
func hasCacheValidator(headers http.Header) bool {
	return headers.Get("if-none-match") != "" || headers.Get("if-modified-since") != ""
}
