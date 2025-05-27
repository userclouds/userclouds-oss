package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/test/testlogtransport"
)

func TestGzip(t *testing.T) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	// TODO (sgarrity 10/23): this is a lazy way to do this but :shrug:
	var s string
	for range 500 {
		s += "hello"
	}
	sp := &s

	// return a long body
	h := BaseMiddleware.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// we use a pointer here so we can edit it later and be lazy :)
		w.Write([]byte(*sp))
	}))

	srv := httptest.NewServer(h)
	defer srv.Close()

	// test without accepting gzip
	r, err := http.NewRequest("GET", srv.URL, nil)
	r.Header.Set("Accept-Encoding", "") // explicitly do not accept gzip
	assert.NoErr(t, err)
	res, err := http.DefaultClient.Do(r)
	assert.NoErr(t, err)
	assert.Equal(t, res.Header.Get("Content-Encoding"), "")
	assert.Equal(t, res.ContentLength, int64(-1)) // not set on large responses per https://github.com/golang/go/issues/23450

	tt.AssertLogsContainString("2500B -> 2500B")

	// accept gzip and see what happens
	r, err = http.NewRequest("GET", srv.URL, nil)
	assert.NoErr(t, err)
	r.Header.Set("Accept-Encoding", "gzip") // explicitly accept gzip
	res, err = http.DefaultClient.Do(r)
	assert.NoErr(t, err)
	assert.Equal(t, res.Header.Get("Content-Encoding"), "gzip")
	assert.Equal(t, res.ContentLength, int64(47))

	tt.AssertLogsContainString("2500B -> 47B")

	// test that a really small body is not gzipped even when accepted
	*sp = "hello"
	r, err = http.NewRequest("GET", srv.URL, nil)
	assert.NoErr(t, err)
	r.Header.Set("Accept-Encoding", "gzip")
	res, err = http.DefaultClient.Do(r)
	assert.NoErr(t, err)
	assert.Equal(t, res.Header.Get("Content-Encoding"), "")
	assert.Equal(t, res.ContentLength, int64(5))

	tt.AssertLogsContainString("5B -> 5B")

	// test that a midsize body (50B) is still not gzipped even when accepted
	s = ""
	for range 10 {
		s += "hello"
	}
	*sp = s
	r, err = http.NewRequest("GET", srv.URL, nil)
	assert.NoErr(t, err)
	r.Header.Set("Accept-Encoding", "gzip")
	res, err = http.DefaultClient.Do(r)
	assert.NoErr(t, err)
	assert.Equal(t, res.Header.Get("Content-Encoding"), "")
	assert.Equal(t, res.ContentLength, int64(50))

	tt.AssertLogsContainString("50B -> 50B")

}
