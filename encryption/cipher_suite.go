package encryption

// CipherSuite represents the available cipher suites for encryption
type CipherSuite string

const (
	// AES256GCM represents AES-256 in GCM mode (recommended)
	// Provides authenticated encryption with associated data (AEAD)
	AES256GCM CipherSuite = "AES256GCM"

	// AES256CBC represents AES-256 in CBC mode with PKCS7 padding
	// Traditional block cipher mode, requires explicit IV
	AES256CBC CipherSuite = "AES256CBC"
)



// IsValid checks if the cipher suite is supported
func (c CipherSuite) IsValid() bool {
	switch c {
	case AES256GCM, AES256CBC:
		return true
	default:
		return false
	}
}

// RequiresIV returns true if the cipher suite requires an explicit IV
func (c CipherSuite) RequiresIV() bool {
	return c == AES256CBC
} 