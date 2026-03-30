package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"log/slog"
	"sync"

	"github.com/rarebek/wt"
)

// Compressor defines the interface for message compression.
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	Name() string
}

// GzipCompressor implements gzip compression.
type GzipCompressor struct {
	pool sync.Pool
}

// NewGzipCompressor creates a gzip compressor with writer pooling.
func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{
		pool: sync.Pool{
			New: func() any {
				w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
				return w
			},
		},
	}
}

func (g *GzipCompressor) Name() string { return "gzip" }

func (g *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := g.pool.Get().(*gzip.Writer)
	w.Reset(&buf)

	_, err := w.Write(data)
	if err != nil {
		g.pool.Put(w)
		return nil, err
	}
	if err := w.Close(); err != nil {
		g.pool.Put(w)
		return nil, err
	}
	g.pool.Put(w)
	return buf.Bytes(), nil
}

func (g *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// DeflateCompressor implements deflate compression.
type DeflateCompressor struct {
	pool sync.Pool
}

// NewDeflateCompressor creates a deflate compressor.
func NewDeflateCompressor() *DeflateCompressor {
	return &DeflateCompressor{
		pool: sync.Pool{
			New: func() any {
				w, _ := flate.NewWriter(nil, flate.BestSpeed)
				return w
			},
		},
	}
}

func (d *DeflateCompressor) Name() string { return "deflate" }

func (d *DeflateCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := d.pool.Get().(*flate.Writer)
	w.Reset(&buf)

	_, err := w.Write(data)
	if err != nil {
		d.pool.Put(w)
		return nil, err
	}
	if err := w.Close(); err != nil {
		d.pool.Put(w)
		return nil, err
	}
	d.pool.Put(w)
	return buf.Bytes(), nil
}

func (d *DeflateCompressor) Decompress(data []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	return io.ReadAll(r)
}

// Compress returns middleware that stores a compressor in the context
// for handlers to use when sending large messages.
func Compress(c Compressor, logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx *wt.Context, next wt.HandlerFunc) {
		ctx.Set("_compressor", c)
		logger.Debug("compression enabled", "algorithm", c.Name(), "session", ctx.ID())
		next(ctx)
	}
}

// GetCompressor retrieves the compressor from the context.
// Returns nil if no compression middleware was applied.
func GetCompressor(c *wt.Context) Compressor {
	v, ok := c.Get("_compressor")
	if !ok {
		return nil
	}
	comp, _ := v.(Compressor)
	return comp
}
