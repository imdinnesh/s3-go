# s3-go

**s3-go** is a lightweight, distributed object storage system built in Go. It demonstrates core concepts of distributed storage, specifically **Erasure Coding** (using Reed-Solomon) for fault tolerance and **Web APIs** for easy interaction.

## 🚀 Key Features

-   **Erasure Coding**: Splits data into **4 Data Shards** and **2 Parity Shards**.
-   **Fault Tolerance**: Can reconstruct the original data even if **2 shards (drives) are lost** simultaneously.
-   **Distributed Architecture**: Separate **Storage Nodes** and **Gateway** interact via gRPC.
-   **Self-Healing**: Automatic reconstruction of missing or corrupted shards during download.
-   **HTTP API**: Simple REST endpoints for uploading and downloading files.

## 🏗 Architecture

The system consists of three main components:

1.  **Encoder (`internal/encoder`)**: 
    -   Handles the math behind splitting files into shards and generating parity data.
    -   Uses [Reed-Solomon](https://github.com/klauspost/reedsolomon) coding.
2.  **Storage Nodes (`cmd/storage`)**:
    -   gRPC servers that act as "drives".
    -   Receive chunks of data and write them to the local disk (e.g., `storage_9001/`).
3.  **Gateway (`cmd/gateway`)**:
    -   The orchestration layer exposing an **HTTP API**.
    -   Handles **reed-solomon** encoding (splitting files into shards).
    -   Distributes shards across available storage nodes.
    -   Reconstructs files on download, recovering from node failures seamlessly.

## 🛠 Prerequisites

-   [Go 1.24+](https://go.dev/dl/) installed.
-   [Protoc](https://grpc.io/docs/protoc-installation/) (optional, only if you want to regenerate `proto` files).

## 📦 Installation

Clone the repository and install dependencies:

```bash
git clone https://github.com/imdinnesh/s3-go.git
cd s3-go
go mod tidy
```

## 🎮 Usage

### 1. Start the Storage Nodes
You need to start the storage servers that will hold the data shards. Open 3 terminal tabs and run:

```bash
# Terminal 1
go run cmd/storage/main.go 9001

# Terminal 2
go run cmd/storage/main.go 9002

# Terminal 3
go run cmd/storage/main.go 9003
```

This will create directories `storage_9001`, `storage_9002`, and `storage_9003` to store the actual data chunks.

### 2. Start the Gateway
The Gateway acts as the entry point for your applications. It connects to the storage nodes and exposes an HTTP server on port **8080**.

```bash
# Terminal 4
go run cmd/gateway/main.go
```

**Expected Output:**
```text
✅ Connected to Storage Node at localhost:9001
✅ Connected to Storage Node at localhost:9002
✅ Connected to Storage Node at localhost:9003
🚀 Gateway running on http://localhost:8080
```

### 3. Interact with the System

Now you can upload and download files using `curl` or any HTTP client (like Postman).

#### 📤 Upload a File
Split a file into shards and distribute them across the storage nodes.

```bash
curl -X POST -F "file=@mydata.txt" http://localhost:8080/upload
```
*Replace `mydata.txt` with any file on your system.*

#### 📥 Download a File
Retrieve the file. The Gateway will fetch shards in parallel and reconstruct the original file, even if some nodes are down.

```bash
# Download and save as recovered_data.txt
curl -o recovered_data.txt http://localhost:8080/download/mydata.txt
```

#### 📊 Check System Status
View the health of your storage nodes.

```bash
curl http://localhost:8080/status
```

**Response:**
```json
[
  {"id": 1, "name": "Storage-1", "status": "alive"},
  {"id": 2, "name": "Storage-2", "status": "alive"},
  {"id": 3, "name": "Storage-3", "status": "alive"}
]
```

## 🧪 Simulation / Testing
### Run Local Simulation (No Server Required)
To understand the underlying erasure coding logic without running servers, use the standalone simulation:

```bash
go run cmd/main.go
```
This script encodes a string, **deliberately deletes 2 shards**, and proves it can still reconstruct the data.

### Direct gRPC Client (Optional)
To test a specific storage node directly:
```bash
go run cmd/client/main.go
```

## 🔧 Development

### Generating Protocol Buffers
If you modify `internal/transport/storage.proto`, regenerate the Go code using:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    internal/transport/storage.proto
```
