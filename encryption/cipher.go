package encryption

import "io"

// StreamCipher defines the interface for stream encryption/decryption
type StreamCipher interface {
	// EncryptStream encrypts data from src and writes to dst
	EncryptStream(src io.Reader, dst io.Writer) error
	
	// DecryptStream decrypts data from src and writes to dst
	DecryptStream(src io.Reader, dst io.Writer) error
}

// NewCBCCipher creates a new AES-CBC cipher with the given key and IV
func NewCBCCipher(key, iv []byte) (StreamCipher, error) {
	return &cbcCipher{
		key: key,
		iv:  iv,
	}, nil
}

// NewGCMCipher creates a new AES-GCM cipher with the given key
func NewGCMCipher(key []byte) (StreamCipher, error) {
	return &gcmCipher{
		key: key,
	}, nil
} 