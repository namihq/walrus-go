package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestLargeFileEncryption tests encryption and decryption of large files (10MB)
// for both CBC and GCM modes to ensure they can handle large data streams efficiently
func TestLargeFileEncryption(t *testing.T) {
	// 10MB test data
	size := 10 * 1024 * 1024
	plaintext := make([]byte, size)
	_, err := rand.Read(plaintext)
	if err != nil {
		t.Fatalf("Failed to generate test data: %v", err)
	}

	// Test CBC mode
	t.Run("CBC-10MB", func(t *testing.T) {
		key := make([]byte, 32)
		iv := make([]byte, 16)
		_, err := rand.Read(key)
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}
		_, err = rand.Read(iv)
		if err != nil {
			t.Fatalf("Failed to generate IV: %v", err)
		}

		cipher, err := NewCBCCipher(key, iv)
		if err != nil {
			t.Fatalf("Failed to create CBC cipher: %v", err)
		}

		var encrypted bytes.Buffer
		var decrypted bytes.Buffer

		// Encrypt
		err = cipher.EncryptStream(bytes.NewReader(plaintext), &encrypted)
		if err != nil {
			t.Fatalf("CBC encryption failed: %v", err)
		}

		// Decrypt
		err = cipher.DecryptStream(bytes.NewReader(encrypted.Bytes()), &decrypted)
		if err != nil {
			t.Fatalf("CBC decryption failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted.Bytes()) {
			t.Error("CBC decrypted data doesn't match original")
		}
	})

	// Test GCM mode
	t.Run("GCM-10MB", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}

		cipher, err := NewGCMContentCipher(key)
		if err != nil {
			t.Fatalf("Failed to create GCM cipher: %v", err)
		}

		var encrypted bytes.Buffer
		var decrypted bytes.Buffer

		// Encrypt
		err = cipher.EncryptStream(bytes.NewReader(plaintext), &encrypted)
		if err != nil {
			t.Fatalf("GCM encryption failed: %v", err)
		}

		// Decrypt
		err = cipher.DecryptStream(bytes.NewReader(encrypted.Bytes()), &decrypted)
		if err != nil {
			t.Fatalf("GCM decryption failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted.Bytes()) {
			t.Error("GCM decrypted data doesn't match original")
		}
	})
} 