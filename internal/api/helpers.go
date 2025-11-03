package api

import (
	"io"
	"sync/atomic"
)

// Allow to re-read the body of a request
type Repeat struct {
	reader io.ReaderAt
	offset int64
}

func (p *Repeat) Read(val []byte) (n int, err error) {
	n, err = p.reader.ReadAt(val, p.offset)
	atomic.AddInt64(&p.offset, int64(n))
	return n, err
}

func (p *Repeat) Reset() {
	atomic.StoreInt64(&p.offset, 0)
}

func (p *Repeat) Close() error {
	if p.reader != nil {
		if r, ok := p.reader.(io.Closer); ok {
			return r.Close()
		}
	}
	return nil
}
