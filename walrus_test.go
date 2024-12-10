package walrus_go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testContent = "Hello, Walrus!"
)

// Helper function to create a test client
func newTestClient(t *testing.T) *Client {
	return NewClient()
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

// TestStore tests storing data
func TestStore(t *testing.T) {
	client := newTestClient(t)
	resp, err := client.Store([]byte(testContent), &StoreOptions{Epochs: 1})
	if err != nil {
		t.Fatalf("Failed to store data: %v", err)
	}

	resp.NormalizeBlobResponse()
	if resp.Blob.BlobID == "" {
		t.Error("Store operation failed: received empty blob ID in response")
	}
	if resp.Blob.EndEpoch <= 0 {
		t.Error("Store operation failed: received invalid end epoch (must be positive)")
	}
}

// TestStoreFromReader tests storing data from a reader
func TestStoreFromReader(t *testing.T) {
	client := newTestClient(t)
	reader := strings.NewReader(testContent)

	resp, err := client.StoreFromReader(reader, int64(len(testContent)), &StoreOptions{Epochs: 1})
	if err != nil {
		t.Fatalf("Failed to store data from reader: %v", err)
	}

	resp.NormalizeBlobResponse()
	if resp.Blob.BlobID == "" {
		t.Error("StoreFromReader operation failed: received empty blob ID in response")
	}
}

// TestStoreFromURL tests storing data from a URL
func TestStoreFromURL(t *testing.T) {
	client := newTestClient(t)
	testURL := "https://raw.githubusercontent.com/suiet/walrus-go/main/README.md"

	resp, err := client.StoreFromURL(testURL, &StoreOptions{Epochs: 1})
	if err != nil {
		t.Fatalf("Failed to store data from URL %s: %v", testURL, err)
	}

	resp.NormalizeBlobResponse()
	if resp.Blob.BlobID == "" {
		t.Errorf("StoreFromURL operation failed: received empty blob ID when storing from URL %s", testURL)
	}
}

// TestStoreFile tests storing a file
func TestStoreFile(t *testing.T) {
	client := newTestClient(t)

	tmpfile, err := os.CreateTemp("", "walrus-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary test file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testContent)); err != nil {
		t.Fatalf("Failed to write test content to temporary file: %v", err)
	}
	tmpfile.Close()

	resp, err := client.StoreFile(tmpfile.Name(), &StoreOptions{Epochs: 1})
	if err != nil {
		t.Fatalf("Failed to store file %s: %v", tmpfile.Name(), err)
	}

	resp.NormalizeBlobResponse()
	if resp.Blob.BlobID == "" {
		t.Errorf("StoreFile operation failed: received empty blob ID when storing file %s", tmpfile.Name())
	}
}

// TestHead tests retrieving blob metadata
func TestHead(t *testing.T) {
	client := newTestClient(t)
	blobID := storeTestContent(t, client)

	metadata, err := client.Head(blobID)
	if err != nil {
		t.Fatalf("Failed to retrieve metadata for blob %s: %v", blobID, err)
	}

	if metadata.ContentLength != int64(len(testContent)) {
		t.Errorf("Head operation returned incorrect content length: expected %d bytes, got %d bytes",
			len(testContent), metadata.ContentLength)
	}
	if metadata.ContentType == "" {
		t.Errorf("Head operation failed: received empty content type for blob %s", blobID)
	}
}

// TestRead tests reading blob content
func TestRead(t *testing.T) {
	client := newTestClient(t)
	blobID := storeTestContent(t, client)

	data, err := client.Read(blobID)
	if err != nil {
		t.Fatalf("Failed to read blob %s: %v", blobID, err)
	}

	if string(data) != testContent {
		t.Errorf("Read operation returned incorrect content: expected %q, got %q",
			testContent, string(data))
	}
}

// TestReadToFile tests reading blob to a file
func TestReadToFile(t *testing.T) {
	client := newTestClient(t)
	blobID := storeTestContent(t, client)

	tmpfile, err := os.CreateTemp("", "walrus-read-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	if err := client.ReadToFile(blobID, tmpfile.Name()); err != nil {
		t.Fatalf("Failed to read blob %s to file %s: %v", blobID, tmpfile.Name(), err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read content from output file %s: %v", tmpfile.Name(), err)
	}

	if string(content) != testContent {
		t.Errorf("ReadToFile operation returned incorrect content: expected %q, got %q",
			testContent, string(content))
	}
}

// TestReadToReader tests reading blob to an io.Reader
func TestReadToReader(t *testing.T) {
	client := newTestClient(t)
	blobID := storeTestContent(t, client)

	reader, err := client.ReadToReader(blobID)
	if err != nil {
		t.Fatalf("Failed to get reader for blob %s: %v", blobID, err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("Failed to read content from reader for blob %s: %v", blobID, err)
	}

	if buf.String() != testContent {
		t.Errorf("ReadToReader operation returned incorrect content: expected %q, got %q",
			testContent, buf.String())
	}
}

// TestGetAPISpec tests retrieving API specifications
func TestGetAPISpec(t *testing.T) {
	client := newTestClient(t)

	// Test aggregator API spec
	aggSpec, err := client.GetAPISpec(true)
	if err != nil {
		t.Fatalf("Failed to retrieve aggregator API specification: %v", err)
	}
	if len(aggSpec) == 0 {
		t.Error("GetAPISpec operation failed: received empty aggregator API specification")
	}

	// Test publisher API spec
	pubSpec, err := client.GetAPISpec(false)
	if err != nil {
		t.Fatalf("Failed to retrieve publisher API specification: %v", err)
	}
	if len(pubSpec) == 0 {
		t.Error("GetAPISpec operation failed: received empty publisher API specification")
	}
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
		t.Error("Response normalization failed for NewlyCreated response: incorrect blob ID or end epoch")
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
		t.Error("Response normalization failed for AlreadyCertified response: incorrect blob ID or end epoch")
	}
}

// Example usage of the client
func ExampleClient_Store() {
	client := NewClient()
	resp, err := client.Store([]byte("Hello, World!"), &StoreOptions{Epochs: 1})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	resp.NormalizeBlobResponse()
	fmt.Printf("Stored blob with ID: %s\n", resp.Blob.BlobID)
}

// Add new test for retry configuration
func TestWithRetryConfig(t *testing.T) {
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	client := NewClient(
		WithRetryConfig(maxRetries, retryDelay),
	)

	if client.retryConfig.MaxRetries != maxRetries {
		t.Errorf("Expected MaxRetries to be %d, got %d", maxRetries, client.retryConfig.MaxRetries)
	}

	if client.retryConfig.RetryDelay != retryDelay {
		t.Errorf("Expected RetryDelay to be %v, got %v", retryDelay, client.retryConfig.RetryDelay)
	}
}

// Add test for retry functionality
func TestRetryLogic(t *testing.T) {
	// Create a test server that fails first N-1 times and succeeds on Nth attempt
	attemptCount := 0
	maxAttempts := 3
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < maxAttempts {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// Create client with retry config
	client := NewClient(
		WithRetryConfig(maxAttempts, 10*time.Millisecond),
		WithAggregatorURLs([]string{server.URL}),
	)

	// Test read operation with retry
	data, err := client.Read("test-blob")
	if err != nil {
		t.Fatalf("Expected successful read after retries, got error: %v", err)
	}

	if string(data) != "success" {
		t.Errorf("Expected response 'success', got '%s'", string(data))
	}

	if attemptCount != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attemptCount)
	}
}

// Add test for multiple endpoints
func TestMultipleEndpoints(t *testing.T) {
	// Create two test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success from server 2"))
	}))
	defer server2.Close()

	// Create client with multiple endpoints
	client := NewClient(
		WithRetryConfig(2, 10*time.Millisecond),
		WithAggregatorURLs([]string{server1.URL, server2.URL}),
	)

	// Test read operation with endpoint failover
	data, err := client.Read("test-blob")
	if err != nil {
		t.Fatalf("Expected successful read from second server, got error: %v", err)
	}

	if string(data) != "success from server 2" {
		t.Errorf("Expected response 'success from server 2', got '%s'", string(data))
	}
}

// Update TestNewClient to include retry config check
func TestNewClient(t *testing.T) {
	client := NewClient()

	// Check default retry configuration
	if client.retryConfig.MaxRetries != 5 {
		t.Errorf("Expected default MaxRetries to be 5, got %d", client.retryConfig.MaxRetries)
	}

	if client.retryConfig.RetryDelay != 500*time.Millisecond {
		t.Errorf("Expected default RetryDelay to be 500ms, got %v", client.retryConfig.RetryDelay)
	}

	// ... (keep existing URL checks)
}

// Add test for request body preservation during retries
func TestRequestBodyPreservation(t *testing.T) {
	attemptCount := 0
	maxAttempts := 2
	expectedBody := "test content"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}
		if string(body) != expectedBody {
			t.Errorf("Expected body '%s', got '%s'", expectedBody, string(body))
		}

		attemptCount++
		if attemptCount < maxAttempts {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(StoreResponse{
			Blob: BlobInfo{BlobID: "test-id", EndEpoch: 100},
		})
	}))
	defer server.Close()

	client := NewClient(
		WithRetryConfig(maxAttempts, 10*time.Millisecond),
		WithPublisherURLs([]string{server.URL}),
	)

	_, err := client.Store([]byte(expectedBody), nil)
	if err != nil {
		t.Fatalf("Expected successful store after retries, got error: %v", err)
	}

	if attemptCount != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attemptCount)
	}
}
