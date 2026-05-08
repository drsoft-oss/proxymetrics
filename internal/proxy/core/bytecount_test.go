package core_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/proxy/core"
)

func TestCountingReader_HappyPath(t *testing.T) {
	src := strings.NewReader("hello world")
	cr := core.NewCountingReader(src)
	var buf bytes.Buffer
	n, err := io.Copy(&buf, cr)
	if err != nil {
		t.Fatal(err)
	}
	if n != 11 || cr.Count() != 11 {
		t.Fatalf("count: io.Copy=%d, cr.Count=%d", n, cr.Count())
	}
}

func TestCountingReader_PartialOnError(t *testing.T) {
	src := &erroringReader{data: []byte("abcdef"), errAfter: 3}
	cr := core.NewCountingReader(src)
	_, err := io.ReadAll(cr)
	if err == nil {
		t.Fatal("want error")
	}
	if cr.Count() != 3 {
		t.Fatalf("count on partial: got %d, want 3", cr.Count())
	}
}

type erroringReader struct {
	data     []byte
	pos      int
	errAfter int
}

func (e *erroringReader) Read(p []byte) (int, error) {
	if e.pos >= e.errAfter {
		return 0, errors.New("boom")
	}
	n := copy(p, e.data[e.pos:e.errAfter])
	e.pos += n
	return n, nil
}

func TestCountingWriter_Counts(t *testing.T) {
	var dst bytes.Buffer
	cw := core.NewCountingWriter(&dst)
	cw.Write([]byte("foo"))
	cw.Write([]byte("barbaz"))
	if cw.Count() != 9 || dst.Len() != 9 {
		t.Fatalf("count: cw=%d, dst=%d", cw.Count(), dst.Len())
	}
}
