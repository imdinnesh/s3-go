# s3-go

**s3-go** is a lightweight, distributed object storage system built in Go. It demonstrates core concepts of distributed storage, specifically **Erasure Coding** (using Reed-Solomon) for fault tolerance and **gRPC** for efficient network communication.

## 🚀 Key Features

-   **Erasure Coding**: Splits data into **4 Data Shards** and **2 Parity Shards**.
-   **Fault Tolerance**: Can reconstruct the original data even if **2 shards (drives) are lost** simultaneously.
-   **Distributed Architecture**: Separate **Storage Nodes** and **Client** interact via gRPC.
-   **Self-Healing**: Automatic reconstruction of missing or corrupted shards.

## 🏗 Architecture

The system consists of three main components:

1.  **Encoder (`internal/encoder`)**: 
    -   Handles the math behind splitting files into shards and generating parity data.
    -   Uses [Reed-Solomon](https://github.com/klauspost/reedsolomon) coding.
2.  **Storage Node (`cmd/storage`)**:
    -   A gRPC server that acts as a "drive".
    -   Receives chunks of data and writes them to the local disk (e.g., `storage_9001/`).
3.  **Gateway (`cmd/gateway`)**:
    -   The orchestration layer.
    -   Handles **reed-solomon** encoding (splitting files into shards).
    -   Distributes shards across available storage nodes.
4.  **Client (`cmd/client`)**:
    -   (Optional) Can interactions with the system, though the Gateway currently handles upload logic directly for demonstration.

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

### 1. Run the Local Simulation (Demo)

The easiest way to understand the system is to run the standalone simulation. This program:
1.  Encodes a simple string.
2.  **Deliberately deletes** 2 shards to simulate disk failure.
3.  Reconstructs the original data from the remaining shards.

```bash
go run cmd/main.go
```

**Expected Output:**
```text
Original: This is a distributed system that never loses data!
Encoded into 6 shards.
⚠️  Disaster! Deleting Shard 0 and Shard 5...
🚑 Attempting reconstruction...
✅ Reconstruction successful!
Recovered: This is a distributed system that never loses data!
```

### 2. Running a Distributed Storage Node

You can run an actual storage server that listens on a TCP port.

```bash
# Start a storage node on port 9001
go run cmd/storage/main.go 9001
```

This will create a directory named `storage_9001` in your current folder where chunks will be saved.

### 3. Running the Client

Open a new terminal window to run the client. This will verify connection to the storage node and upload a test chunk.

```bash
go run cmd/client/main.go
```

**Expected Output (Client):**
```text
Server Response: Stored successfully (Success: true)
```

**Expected Output (Server):**
```text
📥 Received chunk: test_file_shard_1 (40 bytes)
✅ Saved to storage_9001/test_file_shard_1
```

### 3. Running the Gateway (Distributed Upload)

The **Gateway** acts as the smart client that splits files and distributes them.

1.  **Start 3 Storage Nodes** (in separate terminals):
    ```bash
    go run cmd/storage/main.go 9001 &
    go run cmd/storage/main.go 9002 &
    go run cmd/storage/main.go 9003 &
    ```

2.  **Run the Gateway**:
    ```bash
    go run cmd/gateway/main.go
    ```

**Expected Output (Gateway):**
```text
✅ Connected to Storage Node at localhost:9001
✅ Connected to Storage Node at localhost:9002
✅ Connected to Storage Node at localhost:9003

📤 Uploading secret_plans.txt (94 bytes)...
🪓 File split into 6 shards.
🚀 Sent Shard 0 (47 bytes) -> Node 0
🚀 Sent Shard 1 (47 bytes) -> Node 1
🚀 Sent Shard 2 (47 bytes) -> Node 2
🚀 Sent Shard 3 (47 bytes) -> Node 0
...
✅ Upload Complete!
```

## 🔧 Development

### Generating Protocol Buffers

If you modify `internal/transport/storage.proto`, regenerate the Go code using:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    internal/transport/storage.proto
```
