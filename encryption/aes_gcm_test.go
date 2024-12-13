package encryption

import (
	"bytes"
	"crypto/rand"
	"io"
	"strings"
	"testing"
)

func TestGCMCipher(t *testing.T) {
	// Test cases with different data sizes
	testSizes := []int{
		16,     // Small data
		1024,   // 1KB
		65536,  // 64KB
	}

	for _, size := range testSizes {
		t.Run(formatTestName(size), func(t *testing.T) {
			// Generate random test data
			plaintext := make([]byte, size)
			rand.Read(plaintext)

			// Create cipher
			key := make([]byte, 32)
			rand.Read(key)

			cipher, err := NewGCMContentCipher(key)
			if err != nil {
				t.Fatalf("Failed to create GCM cipher: %v", err)
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

func TestGCMCipherErrors(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		errorMsg string
	}{
		{
			name:     "invalid key size",
			key:      make([]byte, 15),
			errorMsg: "invalid key size: 15",
		},
		{
			name:     "nil key",
			key:      nil,
			errorMsg: "key cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGCMContentCipher(tt.key)
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

func TestGCMAuthenticationAndTampering(t *testing.T) {
	plaintext := []byte("secret message")
	key := make([]byte, 32)
	rand.Read(key)

	cipher, err := NewGCMContentCipher(key)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	// Encrypt data
	var encrypted bytes.Buffer
	err = cipher.EncryptStream(bytes.NewReader(plaintext), &encrypted)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Tamper with encrypted data
	encryptedData := encrypted.Bytes()
	encryptedData[len(encryptedData)-1] ^= 0x01 // Flip last bit

	// Try to decrypt tampered data
	var decrypted bytes.Buffer
	err = cipher.DecryptStream(bytes.NewReader(encryptedData), &decrypted)
	if err == nil {
		t.Error("Expected authentication error for tampered data, got none")
	}
}

func TestGCMStreamErrors(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	cipher, err := NewGCMContentCipher(key)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
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
				return cipher.DecryptStream(bytes.NewReader(make([]byte, 32)), failingWriter)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.test(); err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
} 