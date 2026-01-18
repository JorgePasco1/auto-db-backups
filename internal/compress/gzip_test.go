package compress

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGzipCompressor(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()
	require.NotNil(t, compressor)
	assert.Equal(t, gzip.BestCompression, compressor.level)
}

func TestGzipCompressor_Extension(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()
	assert.Equal(t, ".gz", compressor.Extension())
}

func TestGzipCompressor_CompressDecompress_SmallData(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()
	originalData := []byte("Hello, World! This is a test of gzip compression.")

	// Compress
	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Verify it's valid gzip data by decompressing
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_CompressDecompress_EmptyData(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()
	originalData := []byte{}

	// Compress empty data
	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Verify it's valid gzip data
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_CompressDecompress_LargeData(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()

	// Generate 1MB of compressible data (repeated pattern)
	pattern := "This is a test pattern that will repeat. "
	var builder strings.Builder
	for builder.Len() < 1024*1024 {
		builder.WriteString(pattern)
	}
	originalData := []byte(builder.String())

	// Compress
	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Verify compression actually reduced size for compressible data
	assert.Less(t, len(compressedData), len(originalData),
		"Compressed data should be smaller than original for repetitive content")

	// Verify it's valid gzip data
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_CompressDecompress_RandomData(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()

	// Generate random data (incompressible)
	originalData := make([]byte, 10*1024) // 10KB
	_, err := rand.Read(originalData)
	require.NoError(t, err)

	// Compress
	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Random data may actually be larger after compression due to gzip overhead
	// Just verify it's still valid gzip
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_CompressDecompress_BinaryData(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()

	// Create binary data with null bytes and various byte values
	originalData := make([]byte, 256)
	for i := range originalData {
		originalData[i] = byte(i)
	}

	// Compress
	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Verify it's valid gzip data
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_CompressDecompress_SpecialCharacters(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()

	// Data with special characters, unicode, and newlines
	originalData := []byte("Hello\x00World\nLine2\r\nLine3\tTab\x1bEscape" +
		"Unicode: \u0048\u0065\u006c\u006c\u006f" + // "Hello" in unicode escapes
		"Emoji simulation: [*]")

	// Compress
	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Verify it's valid gzip data
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_StreamingBehavior(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()

	// Create a slow reader that simulates streaming data
	originalData := []byte("Streaming test data that will be read progressively.")

	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	// Read in small chunks to test streaming
	var result bytes.Buffer
	buf := make([]byte, 5)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	// Verify the complete compressed data is valid
	gzReader, err := gzip.NewReader(&result)
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressedData)
}

func TestGzipCompressor_CloseReader(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()
	originalData := []byte("Test data")

	reader := compressor.Compress(bytes.NewReader(originalData))

	// Close the reader
	err := reader.Close()
	assert.NoError(t, err)

	// After closing, reading should return an error or EOF
	buf := make([]byte, 10)
	_, err = reader.Read(buf)
	// The pipe is closed, so we expect an error
	assert.Error(t, err)
}

func TestGzipCompressor_MultipleCompressions(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()

	// Compress same data multiple times and verify each is valid
	originalData := []byte("Test data for multiple compressions")

	for i := 0; i < 3; i++ {
		reader := compressor.Compress(bytes.NewReader(originalData))
		compressedData, err := io.ReadAll(reader)
		reader.Close()
		require.NoError(t, err)

		gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
		require.NoError(t, err)

		decompressedData, err := io.ReadAll(gzReader)
		gzReader.Close()
		require.NoError(t, err)

		assert.Equal(t, originalData, decompressedData)
	}
}

func TestGzipCompressor_OutputIsValidGzipHeader(t *testing.T) {
	t.Parallel()

	compressor := NewGzipCompressor()
	originalData := []byte("Test data")

	reader := compressor.Compress(bytes.NewReader(originalData))
	defer reader.Close()

	compressedData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Check gzip magic bytes
	require.GreaterOrEqual(t, len(compressedData), 10, "Gzip data must have at least 10 bytes for header")
	assert.Equal(t, byte(0x1f), compressedData[0], "First magic byte should be 0x1f")
	assert.Equal(t, byte(0x8b), compressedData[1], "Second magic byte should be 0x8b")
	assert.Equal(t, byte(0x08), compressedData[2], "Compression method should be 0x08 (deflate)")
}

// Benchmark tests
func BenchmarkGzipCompressor_SmallData(b *testing.B) {
	compressor := NewGzipCompressor()
	data := []byte("Small test data for benchmarking compression performance.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := compressor.Compress(bytes.NewReader(data))
		io.ReadAll(reader)
		reader.Close()
	}
}

func BenchmarkGzipCompressor_LargeData(b *testing.B) {
	compressor := NewGzipCompressor()

	// 1MB of compressible data
	pattern := "Benchmark pattern that repeats many times. "
	var builder strings.Builder
	for builder.Len() < 1024*1024 {
		builder.WriteString(pattern)
	}
	data := []byte(builder.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := compressor.Compress(bytes.NewReader(data))
		io.ReadAll(reader)
		reader.Close()
	}
}
