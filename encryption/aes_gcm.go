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

	// Read all data from src
	plaintext, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	// Encrypt and write the data
	ciphertext := c.gcm.Seal(nil, nonce, plaintext, nil)
	_, err = dst.Write(ciphertext)
	return err
}

func (c *gcmContentCipher) DecryptStream(src io.Reader, dst io.Writer) error {
	// Read nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(src, nonce); err != nil {
		return err
	}

	// Read the ciphertext
	ciphertext, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	// Decrypt the data
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	// Write the decrypted data
	_, err = dst.Write(plaintext)
	return err
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
