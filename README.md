# unreader

A Go library providing a buffered io.Reader that is meant for applications that need to rewind the bytes read. This differs from an implementation like bufio.Reader which allows you to Peek, but limits unreads to a single byte or rune.