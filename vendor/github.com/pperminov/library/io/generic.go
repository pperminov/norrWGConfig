package libraryIO

import "io"

func ReadAndClose(in io.ReadCloser) ([]byte, error) {
	defer in.Close()
	return io.ReadAll(in)
}
