package compress

import (
	"compress/gzip"
	"io"
)

type GzipCompressor struct {
	level int
}

func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{level: gzip.BestCompression}
}

func (c *GzipCompressor) Compress(r io.Reader) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		gw, err := gzip.NewWriterLevel(pw, c.level)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		_, err = io.Copy(gw, r)
		if err != nil {
			gw.Close()
			pw.CloseWithError(err)
			return
		}

		if err := gw.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}

		pw.Close()
	}()

	return pr
}

func (c *GzipCompressor) Extension() string {
	return ".gz"
}
