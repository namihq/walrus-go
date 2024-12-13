package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// gcmContentCipher implements ContentCipher interface
type gcmContentCipher struct {
	key []byte
	gcm cipher.AEAD
}

func (c *gcmContentCipher) EncryptStream(src io.Reader, dst io.Writer) error {
	// Add validation
	if c.gcm == nil {
		return fmt.Errorf("gcm cipher is not initialized")
	}

	// Generate nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// Write nonce first
	if _, err := dst.Write(nonce); err != nil {
		return err
	}

	// Create buffer for reading chunks
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			ciphertext := c.gcm.Seal(nil, nonce, buf[:n], nil)
			if _, err := dst.Write(ciphertext); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *gcmContentCipher) DecryptStream(src io.Reader, dst io.Writer) error {
	// Read nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(src, nonce); err != nil {
		return err
	}

	// Create buffer for reading chunks
	buf := make([]byte, 32*1024 + c.gcm.Overhead())
	for {
		n, err := src.Read(buf)
		if n > 0 {
			plaintext, err := c.gcm.Open(nil, nonce, buf[:n], nil)
			if err != nil {
				return err
			}
			if _, err := dst.Write(plaintext); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func NewGCMContentCipher(key []byte) (ContentCipher, error) {
	// Validate key
	if key == nil {
		return nil, fmt.Errorf("key cannot be nil")
	}

	// Validate key length
	switch len(key) {
	case 16, 24, 32: // AES-128, AES-192, AES-256
		// Valid key length
	default:
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}

	// Create AES block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Return initialized cipher with both key and gcm
	return &gcmContentCipher{
		key: key,
		gcm: gcm,
	}, nil
}
