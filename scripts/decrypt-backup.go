// Usage: go run scripts/decrypt-backup.go <encrypted-file> <output-file>
// Requires ENCRYPTION_KEY environment variable (base64-encoded 32-byte key)
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <encrypted-file> <output-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Requires ENCRYPTION_KEY environment variable\n")
		os.Exit(1)
	}

	keyBase64 := os.Getenv("ENCRYPTION_KEY")
	if keyBase64 == "" {
		fmt.Fprintln(os.Stderr, "ENCRYPTION_KEY environment variable not set")
		os.Exit(1)
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode key: %v\n", err)
		os.Exit(1)
	}

	if len(key) != 32 {
		fmt.Fprintf(os.Stderr, "Key must be 32 bytes, got %d\n", len(key))
		os.Exit(1)
	}

	inputFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open input: %v\n", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	// Read all encrypted data
	encryptedData, err := io.ReadAll(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	// AES-256-GCM: first 12 bytes are nonce
	if len(encryptedData) < 12 {
		fmt.Fprintln(os.Stderr, "Encrypted data too short")
		os.Exit(1)
	}

	nonce := encryptedData[:12]
	ciphertext := encryptedData[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create cipher: %v\n", err)
		os.Exit(1)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create GCM: %v\n", err)
		os.Exit(1)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decrypt: %v\n", err)
		os.Exit(1)
	}

	outputFile, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	if _, err := outputFile.Write(plaintext); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Decrypted successfully")
}
