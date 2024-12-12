package walrus_go

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var (
	// ErrDecryption indicates a decryption failure, likely due to an incorrect key
	ErrDecryption = errors.New("failed to decrypt data: invalid key or corrupted data")
	// Magic bytes for encryption validation
	magicBytes = []byte("WAL_V1")
)

// EncryptStream encrypts data from src using AES-CTR and writes the encrypted output to dst
func EncryptStream(key []byte, src io.Reader, dst io.Writer) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
    }

    iv := make([]byte, aes.BlockSize)
    if _, err := rand.Read(iv); err != nil {
        return err
    }

    // Write magic bytes first
    if _, err := dst.Write(magicBytes); err != nil {
        return err
    }

    // Write IV after magic bytes
    if _, err := dst.Write(iv); err != nil {
        return err
    }

    stream := cipher.NewCTR(block, iv)
    
    // Encrypt magic bytes verification
    verificationBytes := make([]byte, len(magicBytes))
    stream.XORKeyStream(verificationBytes, magicBytes)
    if _, err := dst.Write(verificationBytes); err != nil {
        return err
    }

    // Reset stream for actual data encryption
    stream = cipher.NewCTR(block, iv)
    writer := &cipher.StreamWriter{S: stream, W: dst}

    // Copy from src to writer, encryption happens automatically during copy
    _, err = io.Copy(writer, src)
    return err
}

// DecryptStream reads AES-CTR encrypted data from src and writes decrypted output to dst
func DecryptStream(key []byte, src io.Reader, dst io.Writer) error {
    // Read and verify magic bytes
    header := make([]byte, len(magicBytes))
    if _, err := io.ReadFull(src, header); err != nil {
        return ErrDecryption
    }
    if string(header) != string(magicBytes) {
        return ErrDecryption
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return err
    }

    iv := make([]byte, aes.BlockSize)
    if _, err := io.ReadFull(src, iv); err != nil {
        return ErrDecryption
    }

    stream := cipher.NewCTR(block, iv)

    // Read and verify encrypted magic bytes
    encryptedVerification := make([]byte, len(magicBytes))
    if _, err := io.ReadFull(src, encryptedVerification); err != nil {
        return ErrDecryption
    }

    // Decrypt verification bytes
    verificationBytes := make([]byte, len(magicBytes))
    stream.XORKeyStream(verificationBytes, encryptedVerification)
    if string(verificationBytes) != string(magicBytes) {
        return ErrDecryption
    }

    // Reset stream for actual data decryption
    stream = cipher.NewCTR(block, iv)
    reader := &cipher.StreamReader{S: stream, R: src}

    // Copy decrypted data from reader to dst
    _, err = io.Copy(dst, reader)
    return err
}
