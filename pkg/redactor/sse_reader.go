package redactor

import (
	"bufio"
	"bytes"
	"context"
	"io"
)

// SSERedactingReader is a reader that intercepts SSE streams and redacts content.
type SSERedactingReader struct {
	rc     io.ReadCloser
	sr     *StreamRedactor
	br     *bufio.Reader
	outBuf bytes.Buffer
}

// NewSSEReader creates a new SSERedactingReader.
func NewSSEReader(ctx context.Context, rc io.ReadCloser, r *Redactor) io.ReadCloser {
	return &SSERedactingReader{
		rc: rc,
		sr: NewStreamRedactor(ctx, r, 100),
		br: bufio.NewReader(rc),
	}
}

func (s *SSERedactingReader) Read(p []byte) (n int, err error) {
	// If we already have buffered output, yield it first
	if s.outBuf.Len() > 0 {
		return s.outBuf.Read(p)
	}

	// Otherwise, read line by line until we have some output or hit EOF/error
	for s.outBuf.Len() == 0 {
		line, err := s.br.ReadBytes('\n')
		
		if len(line) > 0 {
			redacted := s.sr.RedactSSELine(line)
			s.outBuf.Write(redacted)
		}

		if err != nil {
			if err == io.EOF {
				// Flush any pending SSE window buffer on EOF
				flushed := s.sr.Flush()
				if len(flushed) > 0 {
					s.outBuf.Write(flushed)
				}
			}
			// If we managed to produce output after hitting EOF/error, return it first
			if s.outBuf.Len() > 0 {
				n, _ := s.outBuf.Read(p)
				return n, nil // Defer the error to the next Read call
			}
			return 0, err
		}

		// If we successfully processed a line and generated output, break to return
		if s.outBuf.Len() > 0 {
			break
		}
	}

	return s.outBuf.Read(p)
}

func (s *SSERedactingReader) Close() error {
	return s.rc.Close()
}

// WrapSSEReader wraps an io.ReadCloser with an SSERedactingReader
func (r *Redactor) WrapSSEReader(ctx context.Context, rc io.ReadCloser) io.ReadCloser {
	return NewSSEReader(ctx, rc, r)
}
