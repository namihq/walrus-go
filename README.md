# Walrus Go SDK

The **walrus-go** SDK provides a Go client for interacting with the [Walrus](https://github.com/MystenLabs/walrus) HTTP API. Walrus is a decentralized storage system built on the Sui blockchain, allowing you to store and retrieve blobs efficiently.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Getting Started](#getting-started)
  - [Initializing the Client](#initializing-the-client)
  - [Storing Data](#storing-data)
  - [Retrieving Data](#retrieving-data)
  - [Storing and Retrieving Files](#storing-and-retrieving-files)
- [API Reference](#api-reference)
  - [Client](#client)
  - [StoreOptions](#storeoptions)
  - [Methods](#methods)
    - [Store](#store)
    - [StoreFromReader](#StoreFromReader)
    - [StoreFromURL](#storefromurl)
    - [StoreFile](#storefile)
    - [Head](#head)
    - [Read](#read)
    - [ReadToFile](#readtofile)
    - [GetAPISpec](#getapispec)
    - [ReadToReader](#readtoreader)
- [Encryption](#encryption)
  - [Storing Encrypted Data](#storing-encrypted-data)
  - [Retrieving Encrypted Data](#retrieving-encrypted-data)
  - [Encryption Modes](#encryption-modes)
  - [EncryptionOptions](#encryptionoptions)
- [Contributing](#contributing)
- [License](#license)

## Features

- Store data and files on the Walrus Publisher.
- Retrieve data and files from the Walrus Aggregator.
- Supports specifying storage epochs.
- Supports end-to-end encryption using AES-GCM and AES-CBC.
- Handles response parsing and error handling.

## Installation

To install the **walrus-go** SDK, use `go get`:

```bash
go get github.com/suiet/walrus-go
```

Replace `github.com/suiet/walrus-go` with the actual import path of your module.

## Getting Started

Below is a guide to help you start using the **walrus-go** SDK in your Go projects.

### Initializing the Client

First, import the `walrus` package and create a new client instance:

```go
package main

import (
    "github.com/suiet/walrus-go"
)

func main() {
    // Create client with default testnet endpoints
    client := walrus.NewClient()

    // Or customize with specific options
    client := walrus.NewClient(
        walrus.WithAggregatorURLs([]string{"https://custom-aggregator.example.com"}),
        walrus.WithPublisherURLs([]string{"https://custom-publisher.example.com"}),
        walrus.WithHTTPClient(customHTTPClient),
    )

    // Your code here
}
```

By default, the client uses testnet endpoints. You can customize the client using the following options:

- `WithAggregatorURLs(urls []string)`: Set custom aggregator URLs
- `WithPublisherURLs(urls []string)`: Set custom publisher URLs
- `WithHTTPClient(client *http.Client)`: Set a custom HTTP client

### Storing Data

You can store data on the Walrus Publisher using the `Store` method:

```go
// Store data
data := []byte("some string")
resp, err := client.Store(data, &walrus.StoreOptions{Epochs: 1})

if err != nil {
    log.Fatalf("Error storing data: %v", err)
}

// Check response type and handle accordingly
if resp.NewlyCreated != nil {
    blobID := resp.NewlyCreated.BlobObject.BlobID
    fmt.Printf("Stored new blob ID: %s with cost: %d\n",
        blobID, resp.NewlyCreated.Cost)
} else if resp.AlreadyCertified != nil {
    blobID := resp.AlreadyCertified.BlobID
    fmt.Printf("Blob already exists with ID: %s, end epoch: %d\n",
        blobID, resp.AlreadyCertified.EndEpoch)
}
```

### Retrieving Data

Retrieve the stored data using the `Read` method:

```go
// Read data
retrievedData, err := client.Read(blobID, nil)

if err != nil {
    log.Fatalf("Error reading data: %v", err)
}
fmt.Printf("Retrieved data: %s\n", string(retrievedData))
```

### Storing and Retrieving Files

Store a file on the Walrus Publisher:

```go
fileBlobID, err := client.StoreFile("path/to/your/file.txt", &walrus.StoreOptions{Epochs: 5})
if err != nil {
    log.Fatalf("Error storing file: %v", err)
}
fmt.Printf("Stored file blob ID: %s\n", fileBlobID)
```

Retrieve the file and save it locally:

```go
err = client.ReadToFile(fileBlobID, "path/to/save/file.txt")
if err != nil {
    log.Fatalf("Error reading file: %v", err)
}
fmt.Println("File retrieved successfully")
```

### ReadToReader

Retrieves a blob and returns an io.ReadCloser for streaming the content.

```go
func (c *Client) ReadToReader(blobID string) (io.ReadCloser, error)
```

**Parameters:**

- `blobID string`: The blob ID to retrieve.

**Returns:**

- `io.ReadCloser`: A reader containing the blob content. Remember to close it after use.
- `error`: Error if the operation fails.

**Example:**

```go
reader, err := client.ReadToReader("your-blob-id")
if err != nil {
    log.Fatalf("Error getting reader: %v", err)
}
defer reader.Close()

// Use the reader as needed
_, err = io.Copy(os.Stdout, reader)
if err != nil {
    log.Fatalf("Error reading content: %v", err)
}
```

## API Reference

### Client

The `Client` struct is used to interact with the Walrus API.

#### Fields

- `AggregatorURL []string`: The base URLs of the Walrus Aggregators
- `PublisherURL []string`: The base URLs of the Walrus Publishers
- `httpClient *http.Client`: The HTTP client used for requests

#### NewClient

Creates a new `Client` instance with optional configuration.

```go
func NewClient(opts ...ClientOption) *Client
```

**Parameters:**

- `opts ...ClientOption`: Optional configuration functions

**Available Options:**

- `WithAggregatorURLs(urls []string)`: Set custom aggregator URLs
- `WithPublisherURLs(urls []string)`: Set custom publisher URLs
- `WithHTTPClient(client *http.Client)`: Set a custom HTTP client

**Example:**

```go
// Use default testnet endpoints
client := walrus.NewClient()

// Customize specific options
client := walrus.NewClient(
    // optional: walrus.WithAggregatorURLs([]string{"https://custom-aggregator.example.com"}),
    // optional: walrus.WithPublisherURLs([]string{"https://custom-publisher.example.com"}),
)
```

### StoreOptions

Options for storing data.

#### Fields

- `Epochs int`: Number of storage epochs. Determines how long the data is stored.
- `Encryption *EncryptionOptions`: Optional encryption configuration. If provided, data will be encrypted before storage.

### ReadOptions

Options for reading data.

#### Fields

- `Encryption *EncryptionOptions`: Optional decryption configuration. Must be provided with the same key used for encryption to successfully decrypt the data.

### Methods

#### Store

Stores data on the Walrus Publisher and returns detailed response information.

```go
func (c *Client) Store(data []byte, opts *StoreOptions) (*StoreResponse, error)
```

**Parameters:**

- `data []byte`: The data to store.
- `opts *StoreOptions`: Storage options, including:
  - Number of epochs
  - Encryption configuration (optional)

**Returns:**

- `*StoreResponse`: The complete response containing either NewlyCreated or AlreadyCertified information.
- `error`: Error if the operation fails.

#### StoreFromReader

Stores data from an io.Reader on the Walrus Publisher.

```go
func (c *Client) StoreFromReader(reader io.Reader, contentLength int64, opts *StoreOptions) (*StoreResponse, error)
```

**Parameters:**

- `reader io.Reader`: The source to read data from.
- `contentLength int64`: The total size of the data to be stored. Use -1 if unknown.
- `opts *StoreOptions`: Storage options, such as the number of epochs.

**Returns:**

- `*StoreResponse`: The complete response containing either NewlyCreated or AlreadyCertified information.
- `error`: Error if the operation fails.

#### StoreFromURL

Downloads and stores content from a URL on the Walrus Publisher.

```go
func (c *Client) StoreFromURL(sourceURL string, opts *StoreOptions) (*StoreResponse, error)
```

**Parameters:**

- `sourceURL string`: The URL to download content from.
- `opts *StoreOptions`: Storage options, such as the number of epochs.

**Returns:**

- `*StoreResponse`: The complete response containing either NewlyCreated or AlreadyCertified information.
- `error`: Error if the operation fails.

#### StoreFile

Stores a file on the Walrus Publisher.

```go
func (c *Client) StoreFile(filePath string, opts *StoreOptions) (*StoreResponse, error)
```

**Parameters:**

- `filePath string`: Path to the file to store.
- `opts *StoreOptions`: Storage options.

**Returns:**

- `*StoreResponse`: The complete response containing either NewlyCreated or AlreadyCertified information.
- `error`: Error if the operation fails.

#### Head

Retrieves blob metadata from the Walrus Aggregator without downloading the content.

```go
func (c *Client) Head(blobID string) (*BlobMetadata, error)
```

**Parameters:**

- `blobID string`: The blob ID to get metadata for.

**Returns:**

- `*BlobMetadata`: Contains metadata information including:
  - `ContentLength`: Size of the blob in bytes
  - `ContentType`: MIME type of the content
  - `LastModified`: Last modification timestamp
  - `ETag`: Entity tag for cache validation
- `error`: Error if the operation fails.

**Example:**

```go
metadata, err := client.Head("your-blob-id")
if err != nil {
    log.Fatalf("Error getting metadata: %v", err)
}
fmt.Printf("Blob size: %d bytes\n", metadata.ContentLength)
fmt.Printf("Content type: %s\n", metadata.ContentType)
fmt.Printf("Last modified: %s\n", metadata.LastModified)
```

#### Read

Retrieves data from the Walrus Aggregator.

```go
func (c *Client) Read(blobID string, opts *ReadOptions) ([]byte, error)
```

**Parameters:**

- `blobID string`: The blob ID of the data to retrieve.
- `opts *ReadOptions`: Read options, including:
  - Decryption configuration (optional, must match encryption key if data was encrypted)

**Returns:**

- `[]byte`: The retrieved data (decrypted if encryption options were provided).
- `error`: Error if the operation fails.

#### ReadToFile

Retrieves data and saves it to a file.

```go
func (c *Client) ReadToFile(blobID, filePath string, opts *ReadOptions) error
```

**Parameters:**

- `blobID string`: The blob ID of the data to retrieve.
- `filePath string`: Path to save the retrieved file.
- `opts *ReadOptions`: Read options, including:
  - Decryption configuration (optional, must match encryption key if data was encrypted)

**Returns:**

- `error`: Error if the operation fails.

#### GetAPISpec

Retrieves the API specification from the aggregator or publisher.

```go
func (c *Client) GetAPISpec(isAggregator bool) ([]byte, error)
```

**Parameters:**

- `isAggregator bool`: Set to `true` to get the aggregator's API spec; `false` for the publisher.

**Returns:**

- `[]byte`: The API specification data.
- `error`: Error if the operation fails.

#### ReadToReader

Retrieves a blob and returns an io.ReadCloser for streaming the content.

```go
func (c *Client) ReadToReader(blobID string, options *ReadOptions) (io.ReadCloser, error)
```

**Parameters:**

- `blobID string`: The blob ID to retrieve.
- `options *ReadOptions`: Read options, including:
  - Decryption configuration (optional, must match encryption key if data was encrypted)

**Returns:**

- `io.ReadCloser`: A reader containing the blob content. Remember to close it after use.
- `error`: Error if the operation fails.

**Example:**

```go
reader, err := client.ReadToReader("your-blob-id")
if err != nil {
    log.Fatalf("Error getting reader: %v", err)
}
defer reader.Close()

// Use the reader as needed
_, err = io.Copy(os.Stdout, reader)
if err != nil {
    log.Fatalf("Error reading content: %v", err)
}
```

## Encryption

The SDK supports end-to-end encryption using AES in two modes: GCM (recommended) and CBC.

### Storing Encrypted Data

```go
// Generate a random key for encryption
key := make([]byte, 32) // AES-256 key
rand.Read(key)

// Using GCM mode (recommended, provides authentication)
resp, err := client.Store(data, &walrus.StoreOptions{
    Epochs: 1,
    Encryption: &walrus.EncryptionOptions{
        Key: key,
        Mode: "GCM", // Default mode if not specified
    },
})

// Or using CBC mode (requires IV)
iv := make([]byte, 16)
rand.Read(iv)
resp, err := client.Store(data, &walrus.StoreOptions{
    Epochs: 1,
    Encryption: &walrus.EncryptionOptions{
        Key:  key,
        Mode: "CBC",
        IV:   iv,
    },
})
```

### Retrieving Encrypted Data

```go
// Using GCM mode
retrievedData, err := client.Read(blobID, &walrus.ReadOptions{
    Encryption: &walrus.EncryptionOptions{
        Key: key,
        Mode: "GCM",
    },
})

// Using CBC mode (must provide the same IV used for encryption)
retrievedData, err := client.Read(blobID, &walrus.ReadOptions{
    Encryption: &walrus.EncryptionOptions{
        Key:  key,
        Mode: "CBC",
        IV:   iv,
    },
})
```

### Encryption Modes

1. **GCM (Galois/Counter Mode)** - Recommended

   - Provides both confidentiality and authenticity
   - Automatically handles IV/nonce generation
   - No padding required

2. **CBC (Cipher Block Chaining)**
   - Traditional block cipher mode
   - Requires explicit IV
   - Uses PKCS7 padding

> **Security Note**: For CBC mode, never reuse the same key and IV combination.
> Always generate a new random IV for each encryption operation.

### EncryptionOptions

```go
type EncryptionOptions struct {
    // The encryption/decryption key
    // Should be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256
    Key []byte

    // The encryption mode: "GCM" (default) or "CBC"
    Mode string

    // Initialization Vector, required for CBC mode
    // Must be 16 bytes
    IV []byte
}
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/your-feature`).
3. Commit your changes (`git commit -am 'Add new feature'`).
4. Push to the branch (`git push origin feature/your-feature`).
5. Open a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

# Contact

For any questions or support, please open an issue on the GitHub repository.
