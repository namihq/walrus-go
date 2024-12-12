package walrus_go

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
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

    // Write IV to output stream for later decryption
    if _, err := dst.Write(iv); err != nil {
        return err
    }

    stream := cipher.NewCTR(block, iv)
    writer := &cipher.StreamWriter{S: stream, W: dst}

    // Copy from src to writer, encryption happens automatically during copy
    _, err = io.Copy(writer, src)
    return err
}

// DecryptStream reads AES-CTR encrypted data from src and writes decrypted output to dst
func DecryptStream(key []byte, src io.Reader, dst io.Writer) error {
    block, err := aes.NewCipher(key)
    if err != nil {
        return err
    }

    iv := make([]byte, aes.BlockSize)
    if _, err := io.ReadFull(src, iv); err != nil {
        return err
    }

    stream := cipher.NewCTR(block, iv)
    reader := &cipher.StreamReader{S: stream, R: src}

    // Copy decrypted data from reader to dst
    _, err = io.Copy(dst, reader)
    return err
}
