package core

import (
	"bytes"
	"io"
	"strconv"
	"sync"
)

// StatusPeekReader passes its underlying reader through unchanged, but parses the
// first line as an HTTP status line and invokes onStatus(code) the first time we
// have enough bytes to decide. If the input is not HTTP, onStatus is never called.
type StatusPeekReader struct {
	src      io.Reader
	onStatus func(code int)

	mu      sync.Mutex
	buf     []byte // first-line buffer; capped at 256 bytes
	decided bool
	maxPeek int
}

func NewStatusPeekReader(src io.Reader, onStatus func(code int)) *StatusPeekReader {
	return &StatusPeekReader{src: src, onStatus: onStatus, maxPeek: 256}
}

func (s *StatusPeekReader) Read(p []byte) (int, error) {
	n, err := s.src.Read(p)
	if n > 0 && !s.decided {
		s.mu.Lock()
		s.buf = append(s.buf, p[:n]...)
		if i := bytes.IndexByte(s.buf, '\n'); i >= 0 || len(s.buf) >= s.maxPeek {
			s.decide()
		}
		s.mu.Unlock()
	}
	return n, err
}

// decide must be called with mu held.
func (s *StatusPeekReader) decide() {
	s.decided = true
	line := s.buf
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = bytes.TrimRight(line, "\r")

	// Expect "HTTP/x.y CODE ...".
	parts := bytes.SplitN(line, []byte{' '}, 3)
	if len(parts) < 2 {
		return
	}
	if !bytes.HasPrefix(parts[0], []byte("HTTP/")) {
		return
	}
	code, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return
	}
	if s.onStatus != nil {
		s.onStatus(code)
	}
}
