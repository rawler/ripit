package main

import (
	"io"
	"os"
)

var (
	SEEK_SET = os.SEEK_SET
	SEEK_CUR = os.SEEK_CUR
	SEEK_END = os.SEEK_END
)

type limitedReadSeeker struct {
	underlying io.ReadSeeker
	start, end int64
	pos        int64
}

func (l *limitedReadSeeker) Read(p []byte) (n int, err error) {
	remaining := l.end - l.pos
	if remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > remaining {
		p = p[0:remaining]
	}
	n, err = l.underlying.Read(p)
	l.pos += int64(n)
	return
}

func (l *limitedReadSeeker) Seek(offset int64, whence int) (pos int64, err error) {
	switch whence {
	case os.SEEK_SET:
		offset = l.start + offset
	case os.SEEK_CUR:
		offset = l.pos + offset
	case os.SEEK_END:
		offset = l.end + offset
	default:
		return 0, ErrWhenceUnsupported
	}

	if offset < l.start {
		offset = l.start
	}
	if offset > l.end {
		offset = l.end
	}
	if offset == l.pos {
		return offset - l.start, nil
	}
	pos, err = l.underlying.Seek(offset, SEEK_SET)
	l.pos = pos
	return pos - l.start, err
}

func LimitReadSeeker(underlying io.ReadSeeker, limit int64) (io.ReadSeeker, error) {
	pos, err := underlying.Seek(0, SEEK_CUR)
	if err != nil {
		return nil, err
	}

	return &limitedReadSeeker{
		underlying: underlying,
		start:      pos,
		pos:        pos,
		end:        pos + limit,
	}, nil
}
