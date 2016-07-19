package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

// httpSeekReader is a simple SeekReader implementation allowing a HTTPClient to Seek in file,
// transparently issuing new RangeRequests when needed
type httpSeekReader struct {
	url           string
	resp          *http.Response
	offset        int64
	contentLength int64
}

func (r *httpSeekReader) Seek(offset int64, whence int) (pos int64, err error) {
	switch whence {
	case os.SEEK_SET:
		// All OK
	case os.SEEK_CUR:
		offset = r.offset + offset
	default:
		return 0, ErrWhenceUnsupported
	}

	jump := offset - r.offset
	if jump == 0 {
		return offset, nil
	}

	if jump > 0 && jump < 1000 {
		// Optimization for small jumps, probably faster to just skip ahead
		_, err = io.CopyN(ioutil.Discard, r.resp.Body, jump)
	} else {
		if r.resp != nil {
			r.resp.Body.Close()
		}
		if offset < r.contentLength {
			r.resp, err = GetRange(r.url, offset)
		} else {
			// Enter EOF state
			r.resp = nil
		}
	}
	r.offset = offset
	return offset, err
}

func (r *httpSeekReader) Read(p []byte) (n int, err error) {
	// Did seek beyond EOF
	if r.resp == nil {
		return 0, io.EOF
	}
	n, err = r.resp.Body.Read(p)
	r.offset += int64(n)
	return n, err
}

func HTTPSeekReader(url string) (*httpSeekReader, error) {
	resp, err := GetRange(url, 0)
	if err != nil {
		return nil, err
	}
	return &httpSeekReader{
		resp:          resp,
		url:           url,
		contentLength: resp.ContentLength,
	}, nil
}
