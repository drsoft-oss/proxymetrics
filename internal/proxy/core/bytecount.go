package core

import (
	"io"
	"sync/atomic"
)

// CountingReader wraps an io.Reader and atomically counts bytes successfully read.
type CountingReader struct {
	r io.Reader
	n atomic.Int64
}

func NewCountingReader(r io.Reader) *CountingReader { return &CountingReader{r: r} }

func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.n.Add(int64(n))
	}
	return n, err
}

func (c *CountingReader) Count() int64 { return c.n.Load() }

// CountingWriter wraps an io.Writer and atomically counts bytes successfully written.
type CountingWriter struct {
	w io.Writer
	n atomic.Int64
}

func NewCountingWriter(w io.Writer) *CountingWriter { return &CountingWriter{w: w} }

func (c *CountingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 {
		c.n.Add(int64(n))
	}
	return n, err
}

func (c *CountingWriter) Count() int64 { return c.n.Load() }
