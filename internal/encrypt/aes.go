package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	NonceSize = 12 // GCM standard nonce size
	KeySize   = 32 // AES-256
)

type AESEncryptor struct {
	key []byte
}

func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("key must be exactly %d bytes, got %d", KeySize, len(key))
	}
	return &AESEncryptor{key: key}, nil
}

func (e *AESEncryptor) Encrypt(r io.Reader) (io.ReadCloser, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		// Write nonce first (unencrypted, prepended to ciphertext)
		if _, err := pw.Write(nonce); err != nil {
			pw.CloseWithError(err)
			return
		}

		// Read all data (AES-GCM needs complete data for authentication tag)
		plaintext, err := io.ReadAll(r)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to read plaintext: %w", err))
			return
		}

		// Encrypt and write
		ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
		if _, err := pw.Write(ciphertext); err != nil {
			pw.CloseWithError(err)
			return
		}

		pw.Close()
	}()

	return pr, nil
}

func (e *AESEncryptor) Extension() string {
	return ".enc"
}

// Decrypt decrypts data encrypted with Encrypt
func (e *AESEncryptor) Decrypt(r io.Reader) (io.ReadCloser, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Read nonce first
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(r, nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	// Read ciphertext
	ciphertext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	pr, pw := io.Pipe()
	go func() {
		pw.Write(plaintext)
		pw.Close()
	}()

	return pr, nil
}
