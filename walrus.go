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
)

// Client is a client for interacting with the Walrus API
type Client struct {
    AggregatorURL string
    PublisherURL  string
    httpClient    *http.Client
}

// NewClient creates a new Walrus client
func NewClient(aggregatorURL, publisherURL string) *Client {
    return &Client{
        AggregatorURL: aggregatorURL,
        PublisherURL:  publisherURL,
        httpClient:    &http.Client{},
    }
}

// StoreOptions defines options for storing data
type StoreOptions struct {
    Epochs int // Number of storage epochs
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

// Store stores data on the Walrus Publisher and returns the complete store response
func (c *Client) Store(data []byte, opts *StoreOptions) (*StoreResponse, error) {
    urlStr := fmt.Sprintf("%s/v1/store", c.PublisherURL)
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    req, err := http.NewRequest("PUT", urlStr, bytes.NewReader(data))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/octet-stream")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to store data: %s", string(respData))
    }

    var storeResp StoreResponse
    if err := json.Unmarshal(respData, &storeResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    return &storeResp, nil
}

// StoreReader stores data from an io.Reader and returns the complete store response
func (c *Client) StoreReader(reader io.Reader, contentLength int64, opts *StoreOptions) (*StoreResponse, error) {
    urlStr := fmt.Sprintf("%s/v1/store", c.PublisherURL)
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    req, err := http.NewRequest("PUT", urlStr, reader)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/octet-stream")
    if contentLength >= 0 {
        req.ContentLength = contentLength
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to store data: %s", string(respData))
    }

    var storeResp StoreResponse
    if err := json.Unmarshal(respData, &storeResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

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
        return nil, fmt.Errorf("failed to download from URL, status code: %d", resp.StatusCode)
    }

    return c.StoreReader(resp.Body, resp.ContentLength, opts)
}

// StoreFile stores a file and returns the complete store response
func (c *Client) StoreFile(filePath string, opts *StoreOptions) (*StoreResponse, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        return nil, err
    }

    return c.StoreReader(file, stat.Size(), opts)
}

// Read retrieves a blob from the Walrus Aggregator
func (c *Client) Read(blobID string) ([]byte, error) {
    urlStr := fmt.Sprintf("%s/v1/%s", c.AggregatorURL, url.PathEscape(blobID))

    resp, err := c.httpClient.Get(urlStr)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Check for HTTP errors
    if resp.StatusCode != http.StatusOK {
        respData, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to read blob: %s", string(respData))
    }

    return io.ReadAll(resp.Body)
}

// ReadToFile retrieves a blob and writes it to a file
func (c *Client) ReadToFile(blobID, filePath string) error {
    urlStr := fmt.Sprintf("%s/v1/%s", c.AggregatorURL, url.PathEscape(blobID))

    resp, err := c.httpClient.Get(urlStr)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Check for HTTP errors
    if resp.StatusCode != http.StatusOK {
        respData, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to read blob: %s", string(respData))
    }

    // Create the file
    outFile, err := os.Create(filePath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    // Write the response body to the file
    _, err = io.Copy(outFile, resp.Body)
    return err
}

// GetAPISpec retrieves the API specification from the aggregator or publisher
func (c *Client) GetAPISpec(isAggregator bool) ([]byte, error) {
    var baseURL string
    if isAggregator {
        baseURL = c.AggregatorURL
    } else {
        baseURL = c.PublisherURL
    }

    urlStr := fmt.Sprintf("%s/v1/api", baseURL)

    resp, err := c.httpClient.Get(urlStr)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Check for HTTP errors
    if resp.StatusCode != http.StatusOK {
        respData, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to get API spec: %s", string(respData))
    }

    return io.ReadAll(resp.Body)
}
