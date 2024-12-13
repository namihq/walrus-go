package encryption

import (
	"crypto/aes"
	"fmt"
	"io"
)

// ContentCipher defines the interface for content encryption and decryption
type ContentCipher interface {
    // EncryptStream reads plaintext from src and writes encrypted data to dst
    EncryptStream(src io.Reader, dst io.Writer) error

    // DecryptStream reads ciphertext from src and writes decrypted data to dst
    DecryptStream(src io.Reader, dst io.Writer) error
}

// NewCipher 根据加密套件和密钥创建一个新的加密器
func NewCipher(suite CipherSuite, key []byte, iv []byte) (ContentCipher, error) {
    switch suite {
    case AES256GCM:
        return NewGCMContentCipher(key)
    case AES256CBC:
        return NewCBCCipher(key, iv)
    default:
        return nil, ErrUnsupportedCipherSuite
    }
}

// NewCBCCipher creates a new CBC cipher with the given key and IV
func NewCBCCipher(key, iv []byte) (*cbcCipher, error) {
    if key == nil {
        return nil, fmt.Errorf("invalid key size")
    }
    if iv == nil {
        return nil, fmt.Errorf("IV length must equal block size")
    }

    // AES key size must be 16, 24, or 32 bytes
    switch len(key) {
    case 16, 24, 32:
        // valid key size
    default:
        return nil, fmt.Errorf("invalid key size")
    }

    // IV must be 16 bytes for AES
    if len(iv) != aes.BlockSize {
        return nil, fmt.Errorf("IV length must equal block size")
    }

    return &cbcCipher{
        key: key,
        iv:  iv,
    }, nil
}

// NewGCMCipher creates a new AES-GCM cipher with the given key
func NewGCMCipher(key []byte) (*gcmContentCipher, error) {
    return &gcmContentCipher{
        key: key,
    }, nil
}

var ErrUnsupportedCipherSuite = fmt.Errorf("unsupported cipher suite")
