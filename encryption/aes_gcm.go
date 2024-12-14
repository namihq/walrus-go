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

const (
	// Default buffer size (32KB)
	defaultBufferSize = 32 * 1024
)

func (c *gcmContentCipher) EncryptStream(src io.Reader, dst io.Writer) error {
	if c.gcm == nil {
		return fmt.Errorf("gcm cipher is not initialized")
	}

	// Generate a random key for content encryption
	contentKey := make([]byte, 32)
	if _, err := rand.Read(contentKey); err != nil {
		return fmt.Errorf("failed to generate content key: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Write nonce first
	if _, err := dst.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()

	// Start a goroutine to handle encryption
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		
		// Calculate buffer size to account for maximum possible encryption overhead
		buf := make([]byte, defaultBufferSize)
		// Pre-allocate a larger buffer for the ciphertext that includes GCM overhead
		ciphertextBuf := make([]byte, 0, defaultBufferSize+c.gcm.Overhead())
		
		for {
			n, err := src.Read(buf)
			if n > 0 {
				// Encrypt the chunk, using the pre-allocated buffer
				ciphertext := c.gcm.Seal(ciphertextBuf[:0], nonce, buf[:n], nil)
				
				// Write the encrypted chunk
				if _, err := pw.Write(ciphertext); err != nil {
					errCh <- fmt.Errorf("failed to write encrypted chunk: %w", err)
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

	// Copy encrypted data to destination
	if _, err := io.Copy(dst, pr); err != nil {
		return fmt.Errorf("failed to copy encrypted data: %w", err)
	}

	// Check for encryption errors
	if err := <-errCh; err != nil {
		return err
	}

	return nil
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

		// Calculate exact buffer size for one complete encrypted block
		// GCM overhead (16 bytes) + block size (which should match encryption)
		bufSize := defaultBufferSize + c.gcm.Overhead()
		buf := make([]byte, bufSize)

		for {
			// Read exact size of one encrypted block
			n, err := io.ReadFull(src, buf)
			if err == io.ErrUnexpectedEOF {
				// Handle last partial block
				if n > 0 {
					plaintext, err := c.gcm.Open(nil, nonce, buf[:n], nil)
					if err != nil {
						errCh <- fmt.Errorf("failed to decrypt final block: %w", err)
						return
					}
					if _, err := pw.Write(plaintext); err != nil {
						errCh <- fmt.Errorf("failed to write final decrypted block: %w", err)
						return
					}
				}
				errCh <- nil
				return
			}
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to read source: %w", err)
				return
			}

			// Decrypt the complete block
			plaintext, err := c.gcm.Open(nil, nonce, buf[:n], nil)
			if err != nil {
				errCh <- fmt.Errorf("failed to decrypt block: %w", err)
				return
			}

			// Write the decrypted block
			if _, err := pw.Write(plaintext); err != nil {
				errCh <- fmt.Errorf("failed to write decrypted block: %w", err)
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
