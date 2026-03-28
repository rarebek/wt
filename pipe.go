package wt

import (
	"io"
	"sync"
)

// Pipe bidirectionally copies data between two streams.
// Useful for proxying one WebTransport stream to another.
// Returns when either stream closes or encounters an error.
func Pipe(a, b *Stream) error {
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	setErr := func(err error) {
		if err != nil && err != io.EOF {
			errOnce.Do(func() { firstErr = err })
		}
	}

	wg.Add(2)

	// a → b
	go func() {
		defer wg.Done()
		_, err := io.Copy(b, a)
		setErr(err)
		b.Close()
	}()

	// b → a
	go func() {
		defer wg.Done()
		_, err := io.Copy(a, b)
		setErr(err)
		a.Close()
	}()

	wg.Wait()
	return firstErr
}

// PipeRaw bidirectionally copies between an io.ReadWriteCloser and a Stream.
func PipeRaw(rw io.ReadWriteCloser, s *Stream) error {
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	setErr := func(err error) {
		if err != nil && err != io.EOF {
			errOnce.Do(func() { firstErr = err })
		}
	}

	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(s, rw)
		setErr(err)
		s.Close()
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(rw, s)
		setErr(err)
		rw.Close()
	}()

	wg.Wait()
	return firstErr
}
