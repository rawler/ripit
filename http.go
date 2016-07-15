package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	ErrServerRangeUnsupported = errors.New("Server does not support Range-queries")
	ErrWhenceUnsupported      = errors.New("Does only support SEEK_SET")
)

type httpError string

func (e httpError) Error() string {
	return string(e)
}

func GetRange(url string, offset int64) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	AssertOK(err, "Failed to instantiate a Request for %s", url)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if !strings.Contains(resp.Header.Get("Accept-Ranges"), "bytes") {
			return nil, ErrServerRangeUnsupported
		}
		log.Printf("Server responded with 200 on Range-request. Suspicious, but continuing")
		return resp, nil
	case http.StatusPartialContent:
		return resp, nil
	default:
		return nil, httpError(resp.Status)
	}
}

type httpSeekReader struct {
	url  string
	resp *http.Response
}

func (r *httpSeekReader) Seek(offset int64, whence int) (pos int64, err error) {
	if whence != os.SEEK_SET {
		return 0, ErrWhenceUnsupported
	}
	r.resp.Body.Close()
	r.resp, err = GetRange(r.url, offset)
	return offset, err
}

func (r *httpSeekReader) Read(p []byte) (n int, err error) {
	return r.resp.Body.Read(p)
}

func HTTPSeekReader(url string) (*httpSeekReader, error) {
	resp, err := GetRange(url, 0)
	if err != nil {
		return nil, err
	}
	return &httpSeekReader{resp: resp, url: url}, nil
}
