package walrus_go

import (
    "bytes"
    "fmt"
    "io"
    "os"
    "strings"
    "testing"
)

const (
    testAggregatorURL = "https://aggregator.walrus-testnet.walrus.space"
    testPublisherURL  = "https://publisher.walrus-testnet.walrus.space"
    testContent       = "Hello, Walrus!"
)

// Helper function to create a test client
func newTestClient(t *testing.T) *Client {
    return NewClient(testAggregatorURL, testPublisherURL)
}

// Helper function to store test content and return blobID
func storeTestContent(t *testing.T, client *Client) string {
    resp, err := client.Store([]byte(testContent), &StoreOptions{Epochs: 1})
    if err != nil {
        t.Fatalf("Failed to store test content: %v", err)
    }
    resp.NormalizeBlobResponse()
    return resp.Blob.BlobID
}

// TestNewClient tests client initialization
func TestNewClient(t *testing.T) {
    client := newTestClient(t)
    if client.AggregatorURL != testAggregatorURL {
        t.Errorf("Expected aggregator URL %s, got %s", testAggregatorURL, client.AggregatorURL)
    }
    if client.PublisherURL != testPublisherURL {
        t.Errorf("Expected publisher URL %s, got %s", testPublisherURL, client.PublisherURL)
    }
}

// TestStore tests storing data
func TestStore(t *testing.T) {
    client := newTestClient(t)
    resp, err := client.Store([]byte(testContent), &StoreOptions{Epochs: 1})
    if err != nil {
        t.Fatalf("Store failed: %v", err)
    }

    resp.NormalizeBlobResponse()
    if resp.Blob.BlobID == "" {
        t.Error("Expected non-empty blob ID")
    }
    if resp.Blob.EndEpoch <= 0 {
        t.Error("Expected positive end epoch")
    }

    t.Log("Stored blob with ID:", resp.Blob.BlobID)
}

// TestStoreReader tests storing data from a reader
func TestStoreReader(t *testing.T) {
    client := newTestClient(t)
    reader := strings.NewReader(testContent)

    resp, err := client.StoreReader(reader, int64(len(testContent)), &StoreOptions{Epochs: 1})
    if err != nil {
        t.Fatalf("StoreReader failed: %v", err)
    }

    resp.NormalizeBlobResponse()
    if resp.Blob.BlobID == "" {
        t.Error("Expected non-empty blob ID")
    }

    t.Log("Stored blob with ID:", resp.Blob.BlobID)
}

// TestStoreFromURL tests storing data from a URL
func TestStoreFromURL(t *testing.T) {
    client := newTestClient(t)
    // Using a reliable test URL
    testURL := "https://raw.githubusercontent.com/suiet/walrus-go/main/README.md"

    resp, err := client.StoreFromURL(testURL, &StoreOptions{Epochs: 1})
    if err != nil {
        t.Fatalf("StoreFromURL failed: %v", err)
    }

    resp.NormalizeBlobResponse()
    if resp.Blob.BlobID == "" {
        t.Error("Expected non-empty blob ID")
    }
    t.Log("Stored blob with ID:", resp.Blob.BlobID)
}

// TestStoreFile tests storing a file
func TestStoreFile(t *testing.T) {
    client := newTestClient(t)

    // Create a temporary test file
    tmpfile, err := os.CreateTemp("", "walrus-test-*.txt")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(testContent)); err != nil {
        t.Fatalf("Failed to write to temp file: %v", err)
    }
    tmpfile.Close()

    resp, err := client.StoreFile(tmpfile.Name(), &StoreOptions{Epochs: 1})
    if err != nil {
        t.Fatalf("StoreFile failed: %v", err)
    }

    resp.NormalizeBlobResponse()
    if resp.Blob.BlobID == "" {
        t.Error("Expected non-empty blob ID")
    }
}

// TestHead tests retrieving blob metadata
func TestHead(t *testing.T) {
    client := newTestClient(t)
    blobID := storeTestContent(t, client)

    t.Log("Blob ID:", blobID)

    metadata, err := client.Head(blobID)
    if err != nil {
        t.Fatalf("Head failed: %v", err)
    }

    if metadata.ContentLength != int64(len(testContent)) {
        t.Errorf("Expected content length %d, got %d", len(testContent), metadata.ContentLength)
    }
    if metadata.ContentType == "" {
        t.Error("Expected non-empty content type")
    }
}

// TestRead tests reading blob content
func TestRead(t *testing.T) {
    client := newTestClient(t)
    blobID := storeTestContent(t, client)

    data, err := client.Read(blobID)
    if err != nil {
        t.Fatalf("Read failed: %v", err)
    }

    if string(data) != testContent {
        t.Errorf("Expected content %q, got %q", testContent, string(data))
    }

    t.Log("Read content:", string(data))
}

// TestReadToFile tests reading blob to a file
func TestReadToFile(t *testing.T) {
    client := newTestClient(t)
    blobID := storeTestContent(t, client)

    tmpfile, err := os.CreateTemp("", "walrus-read-test-*.txt")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpfile.Name())
    tmpfile.Close()

    if err := client.ReadToFile(blobID, tmpfile.Name()); err != nil {
        t.Fatalf("ReadToFile failed: %v", err)
    }

    content, err := os.ReadFile(tmpfile.Name())
    if err != nil {
        t.Fatalf("Failed to read temp file: %v", err)
    }

    if string(content) != testContent {
        t.Errorf("Expected content %q, got %q", testContent, string(content))
    }

    t.Log("Read content:", string(content))
}

// TestReadToReader tests reading blob to an io.Reader
func TestReadToReader(t *testing.T) {
    client := newTestClient(t)
    blobID := storeTestContent(t, client)

    reader, err := client.ReadToReader(blobID)
    if err != nil {
        t.Fatalf("ReadToReader failed: %v", err)
    }
    defer reader.Close()

    var buf bytes.Buffer
    if _, err := io.Copy(&buf, reader); err != nil {
        t.Fatalf("Failed to read from reader: %v", err)
    }

    if buf.String() != testContent {
        t.Errorf("Expected content %q, got %q", testContent, buf.String())
    }
    t.Log("Read content:", buf.String())
}

// TestGetAPISpec tests retrieving API specifications
func TestGetAPISpec(t *testing.T) {
    client := newTestClient(t)

    // Test aggregator API spec
    aggSpec, err := client.GetAPISpec(true)
    if err != nil {
        t.Fatalf("GetAPISpec(aggregator) failed: %v", err)
    }
    if len(aggSpec) == 0 {
        t.Error("Expected non-empty aggregator API spec")
    }

    // Test publisher API spec
    pubSpec, err := client.GetAPISpec(false)
    if err != nil {
        t.Fatalf("GetAPISpec(publisher) failed: %v", err)
    }
    if len(pubSpec) == 0 {
        t.Error("Expected non-empty publisher API spec")
    }

    t.Log("Aggregator API spec:", string(aggSpec))
}

// TestNormalizeBlobResponse tests response normalization
func TestNormalizeBlobResponse(t *testing.T) {
    // Test with NewlyCreated response
    newResp := &StoreResponse{
        NewlyCreated: &struct {
            BlobObject  BlobObject `json:"blobObject"`
            EncodedSize int        `json:"encodedSize"`
            Cost        int        `json:"cost"`
        }{
            BlobObject: BlobObject{
                BlobID: "test-blob-id",
                Storage: StorageInfo{
                    EndEpoch: 100,
                },
            },
        },
    }
    newResp.NormalizeBlobResponse()
    if newResp.Blob.BlobID != "test-blob-id" || newResp.Blob.EndEpoch != 100 {
        t.Error("Failed to normalize NewlyCreated response")
    }

    // Test with AlreadyCertified response
    certResp := &StoreResponse{
        AlreadyCertified: &struct {
            BlobID   string    `json:"blobId"`
            Event    EventInfo `json:"event"`
            EndEpoch int       `json:"endEpoch"`
        }{
            BlobID:   "test-blob-id",
            EndEpoch: 200,
        },
    }
    certResp.NormalizeBlobResponse()
    if certResp.Blob.BlobID != "test-blob-id" || certResp.Blob.EndEpoch != 200 {
        t.Error("Failed to normalize AlreadyCertified response")
    }

    t.Log("Normalized response:", certResp)
}

// Example usage of the client
func ExampleClient_Store() {
    client := NewClient(testAggregatorURL, testPublisherURL)
    resp, err := client.Store([]byte("Hello, World!"), &StoreOptions{Epochs: 1})
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    resp.NormalizeBlobResponse()
    fmt.Printf("Stored blob with ID: %s\n", resp.Blob.BlobID)
}
