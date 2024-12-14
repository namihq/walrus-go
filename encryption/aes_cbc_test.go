package encryption

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestCBCCipher(t *testing.T) {
	// Test cases with different data sizes
	testSizes := []int{
		16,     // One block
		32,     // Two blocks
		63,     // Not block aligned
		1024,   // 1KB
		65536,  // 64KB
		1048576,   // 1MB
		10485760,  // 10MB
	}

	for _, size := range testSizes {
		t.Run(formatTestName(size), func(t *testing.T) {
			// Generate random test data
			plaintext := make([]byte, size)
			rand.Read(plaintext)

			// Create cipher
			key := make([]byte, 32)
			iv := make([]byte, 16)
			rand.Read(key)
			rand.Read(iv)

			cipher, err := NewCBCCipher(key, iv)
			if err != nil {
				t.Fatalf("Failed to create CBC cipher: %v", err)
			}

			// Test encryption and decryption
			var encrypted bytes.Buffer
			var decrypted bytes.Buffer

			// Encrypt
			err = cipher.EncryptStream(bytes.NewReader(plaintext), &encrypted)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Decrypt
			err = cipher.DecryptStream(bytes.NewReader(encrypted.Bytes()), &decrypted)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify
			if !bytes.Equal(plaintext, decrypted.Bytes()) {
				t.Error("Decrypted data doesn't match original")
			}
		})
	}
}

func TestCBCCipherErrors(t *testing.T) {
	tests := []struct {
		name    string
		key     []byte
		iv      []byte
		wantErr string
	}{
		{
			name:    "invalid key size",
			key:     make([]byte, 15),
			iv:      make([]byte, 16),
			wantErr: "invalid key size",
		},
		{
			name:    "invalid IV size",
			key:     make([]byte, 32),
			iv:      make([]byte, 15),
			wantErr: "IV length must equal block size",
		},
		{
			name:    "nil key",
			key:     nil,
			iv:      make([]byte, 16),
			wantErr: "invalid key size",
		},
		{
			name:    "nil IV",
			key:     make([]byte, 32),
			iv:      nil,
			wantErr: "IV length must equal block size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCBCCipher(tt.key, tt.iv)
			if err == nil {
				t.Error("Expected error but got none")
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Expected error '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestCBCStreamErrors(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	rand.Read(key)
	rand.Read(iv)

	cipher, err := NewCBCCipher(key, iv)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	// Create test data with valid padding
	testData := []byte("test data with padding")
	var encryptedBuf bytes.Buffer
	err = cipher.EncryptStream(bytes.NewReader(testData), &encryptedBuf)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	failingReader := &failingReader{err: io.ErrUnexpectedEOF}
	failingWriter := &failingWriter{err: io.ErrShortWrite}

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "encryption with failing reader",
			test: func() error {
				return cipher.EncryptStream(failingReader, &bytes.Buffer{})
			},
		},
		{
			name: "encryption with failing writer",
			test: func() error {
				return cipher.EncryptStream(bytes.NewReader([]byte("test")), failingWriter)
			},
		},
		{
			name: "decryption with failing reader",
			test: func() error {
				return cipher.DecryptStream(failingReader, &bytes.Buffer{})
			},
		},
		{
			name: "decryption with failing writer",
			test: func() error {
				return cipher.DecryptStream(bytes.NewReader(encryptedBuf.Bytes()), failingWriter)
			},
		},
		{
			name: "decryption with invalid data",
			test: func() error {
				invalidData := make([]byte, 32) // Invalid encrypted data
				return cipher.DecryptStream(bytes.NewReader(invalidData), &bytes.Buffer{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test()
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Helper types and functions
type failingReader struct {
	err error
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

type failingWriter struct {
	err error
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func formatTestName(size int) string {
	switch {
	case size >= 1048576:
		return formatMB(size)
	case size >= 1024:
		return formatKB(size)
	default:
		return formatBytes(size)
	}
}

func formatKB(size int) string {
	return formatBytes(size/1024) + "KB"
}

func formatBytes(size int) string {
	return string(rune(size)) + "B"
}

func formatMB(size int) string {
	return formatBytes(size/1048576) + "MB"
} 