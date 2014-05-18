// Package httpspy provides types for writing net/http middleware and tests.
package httpspy

import (
	"bytes"
	"net/http"
	"sync"
)

// A Spy wraps an http.ResponseWriter and can report the status code written
// after a handler processes a request.
type Spy interface {
	http.ResponseWriter
	// Code returns the code written with WriteHeader() or 200 if WriteHeader()
	// called implicitly on the first call to Write().  Zero is returned if
	// neither Write() nor WriteHeader() has been called.
	Code() int
}

// NewSpy returns a generic, threadsafe Spy implementation.  If w is nil all
// calls to Write succeed.
func NewSpy(w http.ResponseWriter) Spy {
	s := new(simpleSpy)
	s.w = w
	return s
}

// A WriteSpy is a Spy that also reports the bytes written in the response body
// and any transfer error encountered.
type WriteSpy interface {
	Spy
	// Body returns the concatenation of all bytes passed to Write()
	Body() []byte
	// WriteErr returns the first error returned by Write() if any.
	WriteErr() error
}

// NewWriteSpy returns a generic, threadsafe Spy implementation.  If w is nil
// all calls to Write succeed.
func NewWriteSpy(w http.ResponseWriter) WriteSpy {
	s := new(simpleWriteSpy)
	s.simpleSpy = new(simpleSpy)
	s.simpleSpy.w = w
	return s
}

type simpleSpy struct {
	w       http.ResponseWriter
	mut     sync.Mutex
	written bool
	code    int
}

func (s *simpleSpy) Write(p []byte) (int, error) {
	var err error
	n := len(p)
	s.mut.Lock()
	s.written = true
	if s.w != nil {
		n, err = s.w.Write(p)
	}
	s.mut.Unlock()
	return n, err
}

func (s *simpleSpy) Header() http.Header {
	return s.w.Header()
}

func (s *simpleSpy) WriteHeader(code int) {
	s.mut.Lock()
	if !s.written {
		s.written = true
		s.code = code
	}
	s.WriteHeader(code)
	s.mut.Unlock()
}

func (s *simpleSpy) Code() int {
	if s.code == 0 && s.written {
		return http.StatusOK
	}
	return s.code
}

type simpleWriteSpy struct {
	*simpleSpy
	mut sync.Mutex
	buf bytes.Buffer
	err error
}

func (s *simpleWriteSpy) Write(p []byte) (int, error) {
	s.mut.Lock()
	n, err := s.simpleSpy.Write(p)
	if n > 0 {
		s.buf.Write(p[:n])
	}
	s.mut.Unlock()

	if err != nil && s.err == nil {
		s.err = err
	}
	return n, err
}

func (s *simpleWriteSpy) Body() []byte {
	s.mut.Lock()
	p := s.buf.Bytes()
	s.mut.Unlock()
	return p
}

func (s *simpleWriteSpy) WriteErr() error {
	return s.err
}