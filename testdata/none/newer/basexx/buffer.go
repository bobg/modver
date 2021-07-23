package basexx

import "io"

// Buffer can act as a Source or a Dest (but not both at the same time)
// in the case where each byte in a given slice encodes a single digit in the desired base.
// The digits in the buffer are in the expected order:
// namely, most-significant first, least-significant last.
type Buffer struct {
	buf  []byte
	next int
	base Base
}

// NewBuffer produces a Buffer from the given byte slice described by the given Base.
func NewBuffer(buf []byte, base Base) *Buffer {
	return &Buffer{
		buf:  buf,
		next: -1, // "unstarted" sentinel value
		base: base,
	}
}

func (s *Buffer) Read() (int64, error) {
	if s.next < 0 {
		s.next = 0
	}
	if s.next >= len(s.buf) {
		return 0, io.EOF
	}
	dec, err := s.base.Decode([]byte{s.buf[s.next]})
	if err != nil {
		return 0, err
	}
	s.next++
	return dec, nil
}

func (s *Buffer) Prepend(val int64) error {
	if s.next < 0 {
		s.next = len(s.buf)
	}
	if s.next == 0 {
		return io.EOF
	}
	enc, err := s.base.Encode(val)
	if err != nil {
		return err
	}
	if len(enc) != 1 {
		return ErrInvalid
	}
	s.next--
	s.buf[s.next] = enc[0]
	return nil
}

func (s *Buffer) Written() []byte {
	if s.next < 0 {
		return nil
	}
	return s.buf[s.next:]
}

func (s *Buffer) Base() int64 {
	return s.base.N()
}
