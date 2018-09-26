// Package unreader implements a buffered io.Reader that is meant for applications
// that need to rewind the bytes read. This differs from an implementation like
// bufio.Reader which allows you to Peek, but limits unreads to a single byte
// or rune.

package unreader

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/freb/circbuf"
)

type unreader struct {
	cb        *circbuf.Buffer
	rd        io.Reader // reader provided by the client
	bytesRead int64     // read from underlying reader
	cursor    int64     // byte to read next. if cursor < bytesRead, read from buffer

	runeBuf []byte
	runePos int
}

func (u *unreader) Bytes() []byte {
	return u.cb.Bytes()
}

func (u *unreader) BytesRead() int64 {
	return u.bytesRead
}

func (u *unreader) Cursor() int64 {
	return u.cursor
}

// NewUnreader returns an initialized Unreader
func NewUnreader(size int64, r io.Reader) (*unreader, error) {
	cb, err := circbuf.NewBuffer(size)
	if err != nil {
		return nil, err
	}
	ur := &unreader{
		cb:      cb,
		rd:      r,
		runeBuf: make([]byte, utf8.MaxRune),
	}
	return ur, nil
}

func (u *unreader) Unread(c int64) error {
	newCursor := u.cursor - c
	if u.cb.TotalWritten() < (u.bytesRead - newCursor) {
		return fmt.Errorf("cursor < total bytes written")
	}
	if u.cb.Size() < (u.bytesRead - newCursor) {
		return fmt.Errorf("cursor < buffer size")
	}
	u.cursor = newCursor
	return nil
}

// Read functions like a standard io.Reader except if bytes have been
// unread, it will re-read those first.
func (u *unreader) Read(p []byte) (n int, err error) {
	// either return the bytes we can from the buffer, or use underlying reader
	// for simplicity. Don't attempt to maximize bytes returned.
	n = len(p)
	if n == 0 {
		return 0, nil
	}

	// read from reader
	if u.cursor == u.bytesRead {
		n, err = u.rd.Read(p)
		u.cb.Write(p[:n])
		u.bytesRead += int64(n)
		u.cursor += int64(n)
		return n, err
	}

	b := u.cb.Bytes()
	n = copy(p, b[int64(len(b))-(u.bytesRead-u.cursor):])
	u.cursor += int64(n)
	return n, nil
}

// ReadRune reads a sing rule from the underlying reader and returns the
// rune, size, and an error if there was one.
func (u *unreader) ReadRune() (r rune, size int, err error) {
	u.runePos = 0
	for !utf8.FullRune(u.runeBuf[:u.runePos]) && u.runePos < utf8.MaxRune {
		_, err := u.Read(u.runeBuf[u.runePos : u.runePos+1])
		if err != nil {
			return 0, 0, err
		}
		u.runePos += 1
	}
	r, size = utf8.DecodeRune(u.runeBuf[:u.runePos+1])
	return r, size, nil
}

// LastBytes returns the last n buffered bytes, preceding the current
// cursor position.
func (u *unreader) LastBytes(n int) []byte {
	// unread some, then pass in the length of the match you wanted
	b := u.Bytes()
	l := len(b)
	offset := u.bytesRead - u.cursor
	s := 0
	if l-int(offset)-n > 0 {
		s = l - int(offset) - n
	}
	e := int64(l) - offset
	return b[s:e]
}
