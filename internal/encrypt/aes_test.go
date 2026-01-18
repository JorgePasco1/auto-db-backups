package encrypt

import (
	"bytes"
	"crypto/rand"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateValidKey creates a valid 32-byte key for testing
func generateValidKey() []byte {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// generateRandomKey creates a random 32-byte key
func generateRandomKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, KeySize)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestNewAESEncryptor_ValidKey(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)

	require.NoError(t, err)
	require.NotNil(t, encryptor)
	assert.Equal(t, key, encryptor.key)
}

func TestNewAESEncryptor_InvalidKeySize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		keySize int
	}{
		{"empty key", 0},
		{"too short 16 bytes", 16},
		{"too short 24 bytes", 24},
		{"too short 31 bytes", 31},
		{"too long 33 bytes", 33},
		{"too long 64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			key := make([]byte, tt.keySize)
			encryptor, err := NewAESEncryptor(key)

			assert.Error(t, err)
			assert.Nil(t, encryptor)
			assert.Contains(t, err.Error(), "key must be exactly 32 bytes")
		})
	}
}

func TestAESEncryptor_Extension(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	assert.Equal(t, ".enc", encryptor.Extension())
}

func TestAESEncryptor_EncryptDecrypt_SmallData(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	originalData := []byte("Hello, World! This is a test of AES-256-GCM encryption.")

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)
	defer encryptedReader.Close()

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Encrypted data should be different from original
	assert.NotEqual(t, originalData, encryptedData)

	// Decrypt
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)
	defer decryptedReader.Close()

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decryptedData)
}

func TestAESEncryptor_EncryptDecrypt_EmptyData(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	originalData := []byte{}

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)
	defer encryptedReader.Close()

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Even empty data should produce ciphertext (nonce + auth tag)
	assert.Greater(t, len(encryptedData), 0)

	// Decrypt
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)
	defer decryptedReader.Close()

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decryptedData)
}

func TestAESEncryptor_EncryptDecrypt_LargeData(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// Generate 1MB of data
	pattern := "Large data test pattern for encryption. "
	var builder strings.Builder
	for builder.Len() < 1024*1024 {
		builder.WriteString(pattern)
	}
	originalData := []byte(builder.String())

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)
	defer encryptedReader.Close()

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Decrypt
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)
	defer decryptedReader.Close()

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decryptedData)
}

func TestAESEncryptor_EncryptDecrypt_BinaryData(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// Create binary data with all byte values
	originalData := make([]byte, 256)
	for i := range originalData {
		originalData[i] = byte(i)
	}

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)
	defer encryptedReader.Close()

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Decrypt
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)
	defer decryptedReader.Close()

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decryptedData)
}

func TestAESEncryptor_NonceUniqueness(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	originalData := []byte("Same data encrypted multiple times")

	// Encrypt the same data multiple times
	ciphertexts := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
		require.NoError(t, err)

		ciphertexts[i], err = io.ReadAll(encryptedReader)
		encryptedReader.Close()
		require.NoError(t, err)
	}

	// Each ciphertext should be unique (different nonce)
	for i := 0; i < len(ciphertexts); i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			assert.NotEqual(t, ciphertexts[i], ciphertexts[j],
				"Ciphertexts should be different due to unique nonces")
		}
	}

	// But each should decrypt to the same original data
	for i, ct := range ciphertexts {
		decryptedReader, err := encryptor.Decrypt(bytes.NewReader(ct))
		require.NoError(t, err)

		decryptedData, err := io.ReadAll(decryptedReader)
		decryptedReader.Close()
		require.NoError(t, err)

		assert.Equal(t, originalData, decryptedData, "Ciphertext %d should decrypt correctly", i)
	}
}

func TestAESEncryptor_NoncePrepended(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	originalData := []byte("Test data")

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)
	defer encryptedReader.Close()

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Encrypted data should contain nonce (12 bytes) + ciphertext + auth tag
	// Minimum size is NonceSize + len(plaintext) + 16 (auth tag)
	minExpectedSize := NonceSize + len(originalData) + 16
	assert.GreaterOrEqual(t, len(encryptedData), minExpectedSize,
		"Encrypted data should include nonce and auth tag")
}

func TestAESEncryptor_DecryptWrongKey(t *testing.T) {
	t.Parallel()

	key1 := generateValidKey()
	key2 := make([]byte, KeySize)
	for i := range key2 {
		key2[i] = byte(255 - i) // Different from key1
	}

	encryptor1, err := NewAESEncryptor(key1)
	require.NoError(t, err)

	encryptor2, err := NewAESEncryptor(key2)
	require.NoError(t, err)

	originalData := []byte("Secret message")

	// Encrypt with key1
	encryptedReader, err := encryptor1.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	encryptedReader.Close()
	require.NoError(t, err)

	// Try to decrypt with key2 - should fail
	decryptedReader, err := encryptor2.Decrypt(bytes.NewReader(encryptedData))
	if err == nil {
		// Some implementations fail on Decrypt, others on Read
		_, err = io.ReadAll(decryptedReader)
		decryptedReader.Close()
	}

	assert.Error(t, err, "Decryption with wrong key should fail")
}

func TestAESEncryptor_DecryptTamperedData(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	originalData := []byte("Original secret message")

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	encryptedReader.Close()
	require.NoError(t, err)

	// Tamper with the ciphertext (flip a bit in the middle)
	tamperedData := make([]byte, len(encryptedData))
	copy(tamperedData, encryptedData)
	tamperedData[len(tamperedData)/2] ^= 0x01

	// Try to decrypt tampered data - should fail due to auth tag verification
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(tamperedData))
	if err == nil {
		_, err = io.ReadAll(decryptedReader)
		decryptedReader.Close()
	}

	assert.Error(t, err, "Decryption of tampered data should fail")
}

func TestAESEncryptor_DecryptTruncatedData(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	originalData := []byte("Message to be encrypted")

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	encryptedReader.Close()
	require.NoError(t, err)

	// Truncate the data
	truncatedData := encryptedData[:len(encryptedData)-5]

	// Try to decrypt truncated data - should fail
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(truncatedData))
	if err == nil {
		_, err = io.ReadAll(decryptedReader)
		decryptedReader.Close()
	}

	assert.Error(t, err, "Decryption of truncated data should fail")
}

func TestAESEncryptor_DecryptInvalidNonce(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// Data too short to contain a nonce
	shortData := []byte("short")

	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(shortData))
	if err == nil {
		_, err = io.ReadAll(decryptedReader)
		if decryptedReader != nil {
			decryptedReader.Close()
		}
	}

	assert.Error(t, err, "Decryption with data shorter than nonce should fail")
}

func TestAESEncryptor_DecryptEmptyInput(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	emptyData := []byte{}

	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(emptyData))
	if err == nil {
		_, err = io.ReadAll(decryptedReader)
		if decryptedReader != nil {
			decryptedReader.Close()
		}
	}

	assert.Error(t, err, "Decryption of empty data should fail")
}

func TestAESEncryptor_EncryptDecrypt_SpecialCharacters(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// Data with special characters, unicode, null bytes
	originalData := []byte("Hello\x00World\nNew Line\r\nCRLF\tTab\x1bEsc" +
		"Unicode: \u0048\u0065\u006c\u006c\u006f")

	// Encrypt
	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)
	defer encryptedReader.Close()

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Decrypt
	decryptedReader, err := encryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)
	defer decryptedReader.Close()

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decryptedData)
}

func TestAESEncryptor_Constants(t *testing.T) {
	t.Parallel()

	// Verify constants are correct for AES-256-GCM
	assert.Equal(t, 12, NonceSize, "GCM nonce should be 12 bytes")
	assert.Equal(t, 32, KeySize, "AES-256 key should be 32 bytes")
}

func TestAESEncryptor_MultipleOperations(t *testing.T) {
	t.Parallel()

	key := generateValidKey()
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// Perform multiple encrypt/decrypt operations to ensure no state issues
	testData := [][]byte{
		[]byte("First message"),
		[]byte("Second message"),
		[]byte("Third message with more content"),
		[]byte(""),
		[]byte("Final message"),
	}

	for _, original := range testData {
		encryptedReader, err := encryptor.Encrypt(bytes.NewReader(original))
		require.NoError(t, err)

		encryptedData, err := io.ReadAll(encryptedReader)
		encryptedReader.Close()
		require.NoError(t, err)

		decryptedReader, err := encryptor.Decrypt(bytes.NewReader(encryptedData))
		require.NoError(t, err)

		decryptedData, err := io.ReadAll(decryptedReader)
		decryptedReader.Close()
		require.NoError(t, err)

		assert.Equal(t, original, decryptedData)
	}
}

// Benchmark tests
func BenchmarkAESEncryptor_Encrypt_SmallData(b *testing.B) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	encryptor, _ := NewAESEncryptor(key)
	data := []byte("Small test data for benchmarking encryption.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, _ := encryptor.Encrypt(bytes.NewReader(data))
		io.ReadAll(reader)
		reader.Close()
	}
}

func BenchmarkAESEncryptor_Decrypt_SmallData(b *testing.B) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	encryptor, _ := NewAESEncryptor(key)
	data := []byte("Small test data for benchmarking decryption.")

	// Pre-encrypt the data
	reader, _ := encryptor.Encrypt(bytes.NewReader(data))
	encryptedData, _ := io.ReadAll(reader)
	reader.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, _ := encryptor.Decrypt(bytes.NewReader(encryptedData))
		io.ReadAll(reader)
		reader.Close()
	}
}

func BenchmarkAESEncryptor_EncryptDecrypt_LargeData(b *testing.B) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	encryptor, _ := NewAESEncryptor(key)

	// 1MB of data
	data := make([]byte, 1024*1024)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encReader, _ := encryptor.Encrypt(bytes.NewReader(data))
		encData, _ := io.ReadAll(encReader)
		encReader.Close()

		decReader, _ := encryptor.Decrypt(bytes.NewReader(encData))
		io.ReadAll(decReader)
		decReader.Close()
	}
}
