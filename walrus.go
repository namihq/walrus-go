package walrus_go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/suiet/walrus-go/encryption"
)

// RetryConfig defines the retry configuration
type RetryConfig struct {
    MaxRetries int           // Maximum number of retry attempts
    RetryDelay time.Duration // Delay between retries
}

// Client is a client for interacting with the Walrus API
type Client struct {
    AggregatorURL []string
    PublisherURL  []string
    httpClient    *http.Client
    retryConfig   RetryConfig // Add retry configuration
    // MaxUnknownLengthUploadSize specifies the maximum allowed size in bytes for uploads
    // when the content length is not known in advance (i.e., contentLength <= 0).
    // In such cases, the entire content must be read into memory to determine its size,
    // which could potentially cause memory issues with very large uploads.
    // This limit helps prevent memory exhaustion in those scenarios.
    // Default is 5MB.
    MaxUnknownLengthUploadSize int64
}

// ClientOption defines a function type that modifies Client options
type ClientOption func(*Client)

// WithAggregatorURLs sets custom aggregator URLs for the client
func WithAggregatorURLs(urls []string) ClientOption {
    return func(c *Client) {
        if len(urls) > 0 {
            c.AggregatorURL = urls
        }
    }
}

// WithPublisherURLs sets custom publisher URLs for the client
func WithPublisherURLs(urls []string) ClientOption {
    return func(c *Client) {
        if len(urls) > 0 {
            c.PublisherURL = urls
        }
    }
}

// WithHTTPClient sets a custom HTTP client for the Walrus client
func WithHTTPClient(httpClient *http.Client) ClientOption {
    return func(c *Client) {
        if httpClient != nil {
            c.httpClient = httpClient
        }
    }
}

// WithRetryConfig sets the retry configuration for the client
func WithRetryConfig(maxRetries int, retryDelay time.Duration) ClientOption {
    return func(c *Client) {
        c.retryConfig = RetryConfig{
            MaxRetries: maxRetries,
            RetryDelay: retryDelay,
        }
    }
}

// WithMaxUnknownLengthUploadSize sets the maximum allowed size for uploads when content length
// is not known in advance (contentLength <= 0). This applies only when uploading from a reader
// that doesn't provide size information, requiring the entire content to be read into memory first.
// This limit helps prevent potential memory exhaustion in such cases.
// Default is 5MB.
func WithMaxUnknownLengthUploadSize(maxSize int64) ClientOption {
    return func(c *Client) {
        if maxSize > 0 {
            c.MaxUnknownLengthUploadSize = maxSize
        }
    }
}

// NewClient creates a new Walrus client with the specified options
func NewClient(opts ...ClientOption) *Client {
    // Create client with default values
    client := &Client{
        AggregatorURL: DefaultTestnetAggregators,
        PublisherURL:  DefaultTestnetPublishers,
        httpClient:    &http.Client{},
        retryConfig: RetryConfig{
            MaxRetries: 5,                      // Default to 5 retries
            RetryDelay: 500 * time.Millisecond, // Default to 500ms delay
        },
        MaxUnknownLengthUploadSize: 5 * 1024 * 1024, // Default to 5MB
    }

    // Apply all options
    for _, opt := range opts {
        opt(client)
    }

    return client
}

// EncryptionOptions defines the encryption configuration
type EncryptionOptions struct {
    // Key used for encryption/decryption
    Key []byte
    // Mode specifies the encryption mode ("CBC" or "GCM")
    Mode string
    // IV is only required for CBC mode
    IV []byte
}

// StoreOptions defines options for storing data
type StoreOptions struct {
    Epochs int // Number of storage epochs
    // Encryption configuration, if nil encryption is disabled
    Encryption *EncryptionOptions
}

// ReadOptions defines options for reading data
type ReadOptions struct {
    // Encryption configuration for decryption, if nil decryption is disabled
    Encryption *EncryptionOptions
}

// BlobInfo represents the information returned after storing data
type BlobInfo struct {
    BlobID   string `json:"blobId"`
    EndEpoch int    `json:"endEpoch"`
}

// BlobObject represents the blob object information
type BlobObject struct {
    ID              string      `json:"id"`
    StoredEpoch     int         `json:"storedEpoch"`
    BlobID          string      `json:"blobId"`
    Size            int64       `json:"size"`
    ErasureCodeType string      `json:"erasureCodeType"`
    CertifiedEpoch  int         `json:"certifiedEpoch"`
    Storage         StorageInfo `json:"storage"`
}

// StoreResponse represents the unified response for store operations
type StoreResponse struct {
    Blob BlobInfo `json:"blobInfo,omitempty"`

    // For newly created blobs
    NewlyCreated *struct {
        BlobObject  BlobObject `json:"blobObject"`
        EncodedSize int        `json:"encodedSize"`
        Cost        int        `json:"cost"`
    } `json:"newlyCreated,omitempty"`

    // For already certified blobs
    AlreadyCertified *struct {
        BlobID   string    `json:"blobId"`
        Event    EventInfo `json:"event"`
        EndEpoch int       `json:"endEpoch"`
    } `json:"alreadyCertified,omitempty"`
}

// NormalizeBlobResponse is a helper function to normalize the response from the blob service
func (resp *StoreResponse) NormalizeBlobResponse() {
    if resp.AlreadyCertified != nil {
        resp.Blob.BlobID = resp.AlreadyCertified.BlobID
        resp.Blob.EndEpoch = resp.AlreadyCertified.EndEpoch
    }

    if resp.NewlyCreated != nil {
        resp.Blob.BlobID = resp.NewlyCreated.BlobObject.BlobID
        resp.Blob.EndEpoch = resp.NewlyCreated.BlobObject.Storage.EndEpoch
    }
}

// EventInfo represents the certification event information
type EventInfo struct {
    TxDigest string `json:"txDigest"`
    EventSeq string `json:"eventSeq"`
}

// StorageInfo represents the storage information for a blob
type StorageInfo struct {
    ID          string `json:"id"`
    StartEpoch  int    `json:"startEpoch"`
    EndEpoch    int    `json:"endEpoch"`
    StorageSize int    `json:"storageSize"`
}

// BlobMetadata represents the metadata information returned by Head request
type BlobMetadata struct {
    ContentLength int64  `json:"content-length"`
    ContentType   string `json:"content-type"`
    LastModified  string `json:"last-modified"`
    ETag          string `json:"etag"`
}

// Add a helper function to create cipher
func (opts *EncryptionOptions) getCipher() (encryption.StreamCipher, error) {
    if opts == nil || len(opts.Key) == 0 {
        return nil, fmt.Errorf("encryption key is required")
    }

    switch opts.Mode {
    case "CBC":
        if len(opts.IV) == 0 {
            return nil, fmt.Errorf("IV is required for CBC mode")
        }
        return encryption.NewCBCCipher(opts.Key, opts.IV)
    case "GCM", "": // Default to GCM if no mode is specified
        return encryption.NewGCMCipher(opts.Key)
    default:
        return nil, fmt.Errorf("unsupported encryption mode: %s", opts.Mode)
    }
}

// Store stores data on the Walrus Publisher and returns the complete store response
func (c *Client) Store(data []byte, opts *StoreOptions) (*StoreResponse, error) {
    urlStr := "/v1/store"
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    var reader io.Reader = bytes.NewReader(data)

    // If encryption is enabled
    if opts != nil && opts.Encryption != nil {
        cipher, err := opts.Encryption.getCipher()
        if err != nil {
            return nil, fmt.Errorf("failed to create cipher: %w", err)
        }

        var buf bytes.Buffer
        if err := cipher.EncryptStream(bytes.NewReader(data), &buf); err != nil {
            return nil, fmt.Errorf("failed to encrypt data: %w", err)
        }
        reader = &buf
    }

    req, err := http.NewRequest("PUT", urlStr, reader)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/octet-stream")

    resp, err := c.doWithRetry(req, c.PublisherURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var storeResp StoreResponse
    if err := json.Unmarshal(respData, &storeResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    storeResp.NormalizeBlobResponse()

    return &storeResp, nil
}

// StoreFromReader stores data from an io.Reader and returns the complete store response
func (c *Client) StoreFromReader(reader io.Reader, opts *StoreOptions) (*StoreResponse, error) {
    urlStr := "/v1/store"
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    var err error

    // If encryption is enabled
    if opts != nil && opts.Encryption != nil {
        cipher, err := opts.Encryption.getCipher()
        if err != nil {
            return nil, fmt.Errorf("failed to create cipher: %w", err)
        }

        var buf bytes.Buffer
        if err := cipher.EncryptStream(reader, &buf); err != nil {
            return nil, fmt.Errorf("failed to encrypt data: %w", err)
        }
        reader = &buf
    }

    // Create request with the proper reader
    req, err := http.NewRequest("PUT", urlStr, reader)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/octet-stream")

    resp, err := c.doWithRetry(req, c.PublisherURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var storeResp StoreResponse
    if err := json.Unmarshal(respData, &storeResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    storeResp.NormalizeBlobResponse()
    return &storeResp, nil
}

// StoreFromURL downloads and stores content from URL and returns the complete store response
func (c *Client) StoreFromURL(sourceURL string, opts *StoreOptions) (*StoreResponse, error) {
    req, err := http.NewRequest("GET", sourceURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to download from URL: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to download from URL %s: HTTP request returned status code %d, expected 200 OK", sourceURL, resp.StatusCode)
    }

    return c.StoreFromReader(resp.Body, opts)
}

// StoreFile stores a file and returns the complete store response
func (c *Client) StoreFile(filePath string, opts *StoreOptions) (*StoreResponse, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    return c.StoreFromReader(file, opts)
}

// Read retrieves a blob from the Walrus Aggregator
func (c *Client) Read(blobID string, opts *ReadOptions) ([]byte, error) {
    urlStr := fmt.Sprintf("/v1/%s", url.PathEscape(blobID))

    req, err := http.NewRequest(http.MethodGet, urlStr, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.doWithRetry(req, c.AggregatorURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // If decryption is enabled
    if opts != nil && opts.Encryption != nil {
        cipher, err := opts.Encryption.getCipher()
        if err != nil {
            return nil, fmt.Errorf("failed to create cipher: %w", err)
        }

        var decryptedBuf bytes.Buffer
        if err := cipher.DecryptStream(resp.Body, &decryptedBuf); err != nil {
            return nil, fmt.Errorf("failed to decrypt data: %w", err)
        }
        return decryptedBuf.Bytes(), nil
    }

    return io.ReadAll(resp.Body)
}

// ReadToFile retrieves a blob and writes it to a file
func (c *Client) ReadToFile(blobID, filePath string, opts *ReadOptions) error {
    urlStr := fmt.Sprintf("/v1/%s", url.PathEscape(blobID))

    req, err := http.NewRequest(http.MethodGet, urlStr, nil)
    if err != nil {
        return err
    }

    resp, err := c.doWithRetry(req, c.AggregatorURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Create the file
    outFile, err := os.Create(filePath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    // If decryption is enabled
    if opts != nil && opts.Encryption != nil {
        cipher, err := opts.Encryption.getCipher()
        if err != nil {
            return fmt.Errorf("failed to create cipher: %w", err)
        }
        return cipher.DecryptStream(resp.Body, outFile)
    }

    // Write the response body to the file
    _, err = io.Copy(outFile, resp.Body)
    return err
}

// GetAPISpec retrieves the API specification from the aggregator or publisher
func (c *Client) GetAPISpec(isAggregator bool) ([]byte, error) {
    urlStr := "/v1/api"

    req, err := http.NewRequest(http.MethodGet, urlStr, nil)
    if err != nil {
        return nil, err
    }

    urls := c.PublisherURL
    if isAggregator {
        urls = c.AggregatorURL
    }

    resp, err := c.doWithRetry(req, urls)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}

// Head retrieves blob metadata from the Walrus Aggregator without downloading the content
func (c *Client) Head(blobID string) (*BlobMetadata, error) {
    urlStr := fmt.Sprintf("/v1/%s", url.PathEscape(blobID))

    req, err := http.NewRequest(http.MethodHead, urlStr, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create HEAD request: %w", err)
    }

    resp, err := c.doWithRetry(req, c.AggregatorURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    metadata := &BlobMetadata{
        ContentLength: resp.ContentLength,
        ContentType:   resp.Header.Get("Content-Type"),
        LastModified:  resp.Header.Get("Last-Modified"),
        ETag:          resp.Header.Get("ETag"),
    }

    return metadata, nil
}

// ReadToReader retrieves a blob and writes it to the provided io.Writer
func (c *Client) ReadToReader(blobID string, opts *ReadOptions) (io.ReadCloser, error) {
    urlStr := fmt.Sprintf("/v1/%s", url.PathEscape(blobID))

    req, err := http.NewRequest(http.MethodGet, urlStr, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.doWithRetry(req, c.AggregatorURL)
    if err != nil {
        return nil, err
    }

    // If decryption is enabled
    if opts != nil && opts.Encryption != nil {
        cipher, err := opts.Encryption.getCipher()
        if err != nil {
            return nil, fmt.Errorf("failed to create cipher: %w", err)
        }

        var decryptedBuf bytes.Buffer
        if err := cipher.DecryptStream(resp.Body, &decryptedBuf); err != nil {
            return nil, fmt.Errorf("failed to decrypt data: %w", err)
        }
        return io.NopCloser(&decryptedBuf), nil
    }

    return resp.Body, nil
}

// doWithRetry performs an HTTP request with retry logic
func (c *Client) doWithRetry(req *http.Request, urls []string) (*http.Response, error) {
    var lastErr error
    // Calculate total attempts based on retry config and URL count
    totalAttempts := c.retryConfig.MaxRetries + 1
    attemptCount := 0

    // Try URLs in round-robin fashion until max retries reached
    for attemptCount < totalAttempts {
        // Get URL index for this attempt
        urlIndex := attemptCount % len(urls)
        baseURL := urls[urlIndex]

        // Update request URL with current base URL
        req.URL.Host = ""
        req.URL.Scheme = ""
        fullURL := baseURL + req.URL.String()
        req.URL, _ = url.Parse(fullURL)

        // Create a new request for this attempt (since the original body might have been consumed)
        newReq := &http.Request{}
        *newReq = *req
        if req.Body != nil {
            bodyBytes, err := io.ReadAll(req.Body)
            if err != nil {
                return nil, fmt.Errorf("failed to read request body: %w", err)
            }
            req.Body.Close()
            newReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
            req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
        }

        resp, err := c.httpClient.Do(newReq)
        if err == nil && resp.StatusCode == http.StatusOK {
            return resp, nil
        }

        if err != nil {
            lastErr = err
        } else {
            // Attempt to read error message from response body for better error reporting
            errBody, readErr := io.ReadAll(resp.Body)
            resp.Body.Close()
            if readErr == nil && len(errBody) > 0 {
                lastErr = fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(errBody))
            } else {
                lastErr = fmt.Errorf("request failed with status code %d", resp.StatusCode)
            }
        }

        // Sleep before next attempt if not the last attempt
        if attemptCount < totalAttempts-1 {
            time.Sleep(c.retryConfig.RetryDelay)
        }

        attemptCount++
    }

    return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}
