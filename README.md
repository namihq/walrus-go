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
    - [StoreReader](#storereader)
    - [StoreFromURL](#storefromurl)
    - [StoreFile](#storefile)
    - [Head](#head)
    - [Read](#read)
    - [ReadToFile](#readtofile)
    - [GetAPISpec](#getapispec)
- [Contributing](#contributing)
- [License](#license)

## Features

- Store data and files on the Walrus Publisher.
- Retrieve data and files from the Walrus Aggregator.
- Supports specifying storage epochs.
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
    client := walrus.NewClient(
        "https://aggregator.walrus-testnet.walrus.space",
        "https://publisher.walrus-testnet.walrus.space",
    )

    // Your code here
}
```

Replace the URLs with the aggregator and publisher endpoints you wish to use.

### Storing Data

You can store data on the Walrus Publisher using the `Store` method:

```go
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

### Methods

#### Store

Stores data on the Walrus Publisher and returns detailed response information.

```go
func (c *Client) Store(data []byte, opts *StoreOptions) (*StoreResponse, error)
```

**Parameters:**

- `data []byte`: The data to store.
- `opts *StoreOptions`: Storage options, such as the number of epochs.

**Returns:**

- `*StoreResponse`: The complete response containing either NewlyCreated or AlreadyCertified information.
- `error`: Error if the operation fails.

#### StoreReader

Stores data from an io.Reader on the Walrus Publisher.

```go
func (c *Client) StoreReader(reader io.Reader, contentLength int64, opts *StoreOptions) (*StoreResponse, error)
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

### Retrieving Data

Retrieve the stored data using the `Read` method:

```go
retrievedData, err := client.Read(blobID)
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

## API Reference

### Client

The `Client` struct is used to interact with the Walrus API.

#### Fields

- `AggregatorURL string`: The base URL of the Walrus Aggregator.
- `PublisherURL string`: The base URL of the Walrus Publisher.
- `httpClient *http.Client`: The HTTP client used for requests.

#### NewClient

Creates a new `Client` instance.

```go
func NewClient(aggregatorURL, publisherURL string) *Client
```

**Parameters:**

- `aggregatorURL string`: The aggregator's base URL.
- `publisherURL string`: The publisher's base URL.

### StoreOptions

Options for storing data.

#### Fields

- `Epochs int`: Number of storage epochs. Determines how long the data is stored.

### Methods

#### Store

Stores data on the Walrus Publisher.

```go
func (c *Client) Store(data []byte, opts *StoreOptions) (string, error)
```

**Parameters:**

- `data []byte`: The data to store.
- `opts *StoreOptions`: Storage options, such as the number of epochs.

**Returns:**

- `string`: The blob ID of the stored data.
- `error`: Error if the operation fails.

#### StoreFile

Stores a file on the Walrus Publisher.

```go
func (c *Client) StoreFile(filePath string, opts *StoreOptions) (string, error)
```

**Parameters:**

- `filePath string`: Path to the file to store.
- `opts *StoreOptions`: Storage options.

**Returns:**

- `string`: The blob ID of the stored file.
- `error`: Error if the operation fails.

#### Read

Retrieves data from the Walrus Aggregator.

```go
func (c *Client) Read(blobID string) ([]byte, error)
```

**Parameters:**

- `blobID string`: The blob ID of the data to retrieve.

**Returns:**

- `[]byte`: The retrieved data.
- `error`: Error if the operation fails.

#### ReadToFile

Retrieves data and saves it to a file.

```go
func (c *Client) ReadToFile(blobID, filePath string) error
```

**Parameters:**

- `blobID string`: The blob ID of the data to retrieve.
- `filePath string`: Path to save the retrieved file.

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
