package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"bytes"
)

// gcmContentCipher implements ContentCipher interface
type gcmContentCipher struct {
	key []byte
	gcm cipher.AEAD
}

const (
	// Default buffer size (32KB)
	defaultBufferSize = 32 * 1024
)

type gcmEncryptReader struct {
	src    io.Reader
	gcm    cipher.AEAD
	nonce  []byte
	buf    bytes.Buffer
}

type gcmDecryptReader struct {
	src    io.Reader
	gcm    cipher.AEAD
	nonce  []byte
	buf    bytes.Buffer
}

func (c *gcmContentCipher) EncryptStream(src io.Reader, dst io.Writer) error {
	if c.gcm == nil {
		return fmt.Errorf("gcm cipher is not initialized")
	}

	// Generate nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Write nonce first
	if _, err := dst.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}

	reader := &gcmEncryptReader{
		src:    src,
		gcm:    c.gcm,
		nonce:  nonce,
	}

	_, err := io.Copy(dst, reader)
	return err
}

func (r *gcmEncryptReader) Read(data []byte) (int, error) {
	// Similar to cbcEncryptReader but for GCM mode
	n, err := r.src.Read(data)
	if n > 0 {
		r.buf.Write(data[:n])
	}

	if err == io.EOF {
		if r.buf.Len() > 0 {
			plaintext := r.buf.Bytes()
			r.buf.Reset()
			ciphertext := r.gcm.Seal(nil, r.nonce, plaintext, nil)
			copy(data, ciphertext)
			return len(ciphertext), io.EOF
		}
		return 0, io.EOF
	}

	if err != nil {
		return 0, err
	}

	if r.buf.Len() >= defaultBufferSize {
		plaintext := r.buf.Next(defaultBufferSize)
		ciphertext := r.gcm.Seal(nil, r.nonce, plaintext, nil)
		copy(data, ciphertext)
		return len(ciphertext), nil
	}

	return 0, nil
}

func (c *gcmContentCipher) DecryptStream(src io.Reader, dst io.Writer) error {
	// Read the nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(src, nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()

	// Start a goroutine to handle decryption
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()

		// Calculate buffer size to include GCM overhead
		bufSize := defaultBufferSize + c.gcm.Overhead()
		buf := make([]byte, bufSize)

		for {
			n, err := src.Read(buf)
			if n > 0 {
				// Decrypt the chunk
				plaintext, err := c.gcm.Open(nil, nonce, buf[:n], nil)
				if err != nil {
					errCh <- fmt.Errorf("failed to decrypt chunk: %w", err)
					return
				}

				// Write the decrypted chunk
				if _, err := pw.Write(plaintext); err != nil {
					errCh <- fmt.Errorf("failed to write decrypted chunk: %w", err)
					return
				}
			}
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to read source: %w", err)
				return
			}
		}
	}()

	// Copy decrypted data to destination
	if _, err := io.Copy(dst, pr); err != nil {
		return fmt.Errorf("failed to copy decrypted data: %w", err)
	}

	// Check for decryption errors
	if err := <-errCh; err != nil {
		return err
	}

	return nil
}

func NewGCMContentCipher(key []byte) (ContentCipher, error) {
	if key == nil {
		return nil, fmt.Errorf("key cannot be nil")
	}

	switch len(key) {
	case 16, 24, 32: // AES-128, AES-192, AES-256
		// Valid key length
	default:
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &gcmContentCipher{
		key: key,
		gcm: gcm,
	}, nil
}
