package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type gcmCipher struct {
	key []byte
}

func (c *gcmCipher) EncryptStream(src io.Reader, dst io.Writer) error {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
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
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	_, err = dst.Write(ciphertext)
	return err
}

func (c *gcmCipher) DecryptStream(src io.Reader, dst io.Writer) error {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Read nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(src, nonce); err != nil {
		return err
	}

	// Read the ciphertext
	ciphertext, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	// Decrypt the data
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	// Write the decrypted data
	_, err = dst.Write(plaintext)
	return err
}
