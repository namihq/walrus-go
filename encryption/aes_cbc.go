package encryption

import (
    "bytes"
    "crypto/aes"
    "crypto/cipher"
    "io"
)

// PKCS7Padder implements PKCS7 padding
type PKCS7Padder struct {
    blockSize int
}

// Pad adds padding to the input slice according to PKCS7
func (p *PKCS7Padder) Pad(data []byte, size int) ([]byte, error) {
    padding := p.blockSize - (size % p.blockSize)
    padtext := bytes.Repeat([]byte{byte(padding)}, padding)
    return append(data, padtext...), nil
}

// Unpad removes PKCS7 padding from the input slice
func (p *PKCS7Padder) Unpad(data []byte) ([]byte, error) {
    length := len(data)
    if length == 0 {
        return nil, nil
    }

    padding := int(data[length-1])
    return data[:length-padding], nil
}

type cbcEncryptReader struct {
    encrypter cipher.BlockMode
    src       io.Reader
    padder    *PKCS7Padder
    size      int
    buf       bytes.Buffer
}

func (r *cbcEncryptReader) Read(data []byte) (int, error) {
    n, err := r.src.Read(data)
    r.size += n
    blockSize := r.encrypter.BlockSize()
    r.buf.Write(data[:n])

    if err == io.EOF {
        b := make([]byte, getSliceSize(blockSize, r.buf.Len(), len(data)))
        n, err = r.buf.Read(b)
        if err != nil && err != io.EOF {
            return n, err
        }

        if r.buf.Len() == 0 {
            b, err = r.padder.Pad(b[:n], r.size)
            if err != nil {
                return n, err
            }
            n = len(b)
            err = io.EOF
        }

        if n > 0 {
            r.encrypter.CryptBlocks(data, b)
        }
        return n, err
    }

    if err != nil {
        return n, err
    }

    if size := r.buf.Len(); size >= blockSize {
        nBlocks := size / blockSize
        if size > len(data) {
            nBlocks = len(data) / blockSize
        }

        if nBlocks > 0 {
            b := make([]byte, nBlocks*blockSize)
            n, _ = r.buf.Read(b)
            r.encrypter.CryptBlocks(data, b[:n])
        }
    } else {
        n = 0
    }
    return n, nil
}

type cbcDecryptReader struct {
    decrypter cipher.BlockMode
    src       io.Reader
    padder    *PKCS7Padder
    buf       bytes.Buffer
}

func (r *cbcDecryptReader) Read(data []byte) (int, error) {
    n, err := r.src.Read(data)
    blockSize := r.decrypter.BlockSize()
    r.buf.Write(data[:n])

    if err == io.EOF {
        b := make([]byte, getSliceSize(blockSize, r.buf.Len(), len(data)))
        n, err = r.buf.Read(b)
        if err != nil && err != io.EOF {
            return n, err
        }

        if n > 0 {
            r.decrypter.CryptBlocks(data, b)
        }

        if r.buf.Len() == 0 {
            b, err = r.padder.Unpad(data[:n])
            n = len(b)
            if err != nil {
                return n, err
            }
            err = io.EOF
        }
        return n, err
    }

    if err != nil {
        return n, err
    }

    if size := r.buf.Len(); size >= blockSize {
        nBlocks := size / blockSize
        if size > len(data) {
            nBlocks = len(data) / blockSize
        }
        nBlocks -= blockSize

        if nBlocks > 0 {
            b := make([]byte, nBlocks*blockSize)
            n, _ = r.buf.Read(b)
            r.decrypter.CryptBlocks(data, b[:n])
        } else {
            n = 0
        }
    }

    return n, nil
}

func getSliceSize(blockSize, bufSize, dataSize int) int {
    size := bufSize
    if bufSize > dataSize {
        size = dataSize
    }
    size = size - (size % blockSize) - blockSize
    if size <= 0 {
        size = blockSize
    }
    return size
}

type cbcCipher struct {
    key []byte
    iv  []byte
}

// EncryptStreamCBC encrypts data from src using AES-CBC and writes the encrypted output to dst
func (c cbcCipher) EncryptStream(src io.Reader, dst io.Writer) error {
    block, err := aes.NewCipher(c.key)
    if err != nil {
        return err
    }

    // Write IV first
    if _, err := dst.Write(c.iv); err != nil {
        return err
    }

    encrypter := cipher.NewCBCEncrypter(block, c.iv)
    padder := &PKCS7Padder{blockSize: block.BlockSize()}

    reader := &cbcEncryptReader{
        encrypter: encrypter,
        src:       src,
        padder:    padder,
    }

    _, err = io.Copy(dst, reader)
    return err
}

// DecryptStream reads AES-CBC encrypted data from src and writes decrypted output to dst
func (c cbcCipher) DecryptStream(src io.Reader, dst io.Writer) error {
    block, err := aes.NewCipher(c.key)
    if err != nil {
        return err
    }

    // Read IV
    iv := make([]byte, block.BlockSize())
    if _, err := io.ReadFull(src, iv); err != nil {
        return err
    }

    decrypter := cipher.NewCBCDecrypter(block, iv)
    padder := &PKCS7Padder{blockSize: block.BlockSize()}

    reader := &cbcDecryptReader{
        decrypter: decrypter,
        src:       src,
        padder:    padder,
    }

    _, err = io.Copy(dst, reader)
    return err
}
