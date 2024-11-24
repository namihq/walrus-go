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
    BlobID string `json:"blobId"`
}

// NewlyCreatedResponse represents the response when a new blob is created
type NewlyCreatedResponse struct {
    NewlyCreated struct {
        BlobObject struct {
            ID              string `json:"id"`
            StoredEpoch     int    `json:"storedEpoch"`
            BlobID          string `json:"blobId"`
            Size            int    `json:"size"`
            ErasureCodeType string `json:"erasureCodeType"`
            CertifiedEpoch  int    `json:"certifiedEpoch"`
            Storage         struct {
                ID          string `json:"id"`
                StartEpoch  int    `json:"startEpoch"`
                EndEpoch    int    `json:"endEpoch"`
                StorageSize int    `json:"storageSize"`
            } `json:"storage"`
        } `json:"blobObject"`
        EncodedSize int `json:"encodedSize"`
        Cost        int `json:"cost"`
    } `json:"newlyCreated"`
}

// AlreadyCertifiedResponse represents the response when the blob is already certified
type AlreadyCertifiedResponse struct {
    AlreadyCertified struct {
        BlobID string `json:"blobId"`
        Event  struct {
            TxDigest string `json:"txDigest"`
            EventSeq string `json:"eventSeq"`
        } `json:"event"`
        EndEpoch int `json:"endEpoch"`
    } `json:"alreadyCertified"`
}

// Store stores data on the Walrus Publisher and returns the blob ID
func (c *Client) Store(data []byte, opts *StoreOptions) (string, error) {
    urlStr := fmt.Sprintf("%s/v1/store", c.PublisherURL)
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    req, err := http.NewRequest("PUT", urlStr, bytes.NewReader(data))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/octet-stream")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Read and parse the response
    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // Check for HTTP errors
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to store data: %s", string(respData))
    }

    // Try to parse as NewlyCreatedResponse
    var newResp NewlyCreatedResponse
    if err := json.Unmarshal(respData, &newResp); err == nil && newResp.NewlyCreated.BlobObject.BlobID != "" {
        return newResp.NewlyCreated.BlobObject.BlobID, nil
    }

    // Try to parse as AlreadyCertifiedResponse
    var certResp AlreadyCertifiedResponse
    if err := json.Unmarshal(respData, &certResp); err == nil && certResp.AlreadyCertified.BlobID != "" {
        return certResp.AlreadyCertified.BlobID, nil
    }

    return "", fmt.Errorf("unexpected response: %s", string(respData))
}

// StoreReader stores data from an io.Reader on the Walrus Publisher and returns the blob ID.
// The contentLength parameter specifies the total size of the data to be stored.
// If contentLength is unknown, set it to -1 and the request will be sent without Content-Length header.
func (c *Client) StoreReader(reader io.Reader, contentLength int64, opts *StoreOptions) (string, error) {
    // Prepare the URL
    urlStr := fmt.Sprintf("%s/v1/store", c.PublisherURL)
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    // Create new request with the reader as body
    req, err := http.NewRequest("PUT", urlStr, reader)
    if err != nil {
        return "", err
    }

    // Set headers
    req.Header.Set("Content-Type", "application/octet-stream")
    if contentLength >= 0 {
        req.ContentLength = contentLength
    }

    // Send the request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Read and parse the response
    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // Check for HTTP errors
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to store data: %s", string(respData))
    }

    // Try to parse as NewlyCreatedResponse
    var newResp NewlyCreatedResponse
    if err := json.Unmarshal(respData, &newResp); err == nil && newResp.NewlyCreated.BlobObject.BlobID != "" {
        return newResp.NewlyCreated.BlobObject.BlobID, nil
    }

    // Try to parse as AlreadyCertifiedResponse
    var certResp AlreadyCertifiedResponse
    if err := json.Unmarshal(respData, &certResp); err == nil && certResp.AlreadyCertified.BlobID != "" {
        return certResp.AlreadyCertified.BlobID, nil
    }

    return "", fmt.Errorf("unexpected response: %s", string(respData))
}

// StoreFromURL downloads content from the provided URL and stores it on the Walrus Publisher.
// It returns the blob ID of the stored content.
func (c *Client) StoreFromURL(sourceURL string, opts *StoreOptions) (string, error) {
    // Create HTTP request to download the content
    req, err := http.NewRequest("GET", sourceURL, nil)
    if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }

    // Send the request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to download from URL: %w", err)
    }
    defer resp.Body.Close()

    // Check if the download was successful
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to download from URL, status code: %d", resp.StatusCode)
    }

    // Use StoreReader to upload the content
    return c.StoreReader(resp.Body, resp.ContentLength, opts)
}

// StoreFile stores a file on the Walrus Publisher and returns the blob ID
func (c *Client) StoreFile(filePath string, opts *StoreOptions) (string, error) {
    // Open the file
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    // Get file size
    stat, err := file.Stat()
    if err != nil {
        return "", err
    }

    // Prepare the URL
    urlStr := fmt.Sprintf("%s/v1/store", c.PublisherURL)
    if opts != nil && opts.Epochs > 0 {
        urlStr += "?epochs=" + strconv.Itoa(opts.Epochs)
    }

    req, err := http.NewRequest("PUT", urlStr, file)
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/octet-stream")
    req.ContentLength = stat.Size()

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Read and parse the response
    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // Check for HTTP errors
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to store file: %s", string(respData))
    }

    // Try to parse as NewlyCreatedResponse
    var newResp NewlyCreatedResponse
    if err := json.Unmarshal(respData, &newResp); err == nil && newResp.NewlyCreated.BlobObject.BlobID != "" {
        return newResp.NewlyCreated.BlobObject.BlobID, nil
    }

    // Try to parse as AlreadyCertifiedResponse
    var certResp AlreadyCertifiedResponse
    if err := json.Unmarshal(respData, &certResp); err == nil && certResp.AlreadyCertified.BlobID != "" {
        return certResp.AlreadyCertified.BlobID, nil
    }

    return "", fmt.Errorf("unexpected response: %s", string(respData))
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
