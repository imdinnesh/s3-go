package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync" // NEW: Used for fetching shards in parallel
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/imdinnesh/s3-go/internal/encoder"
	pb "github.com/imdinnesh/s3-go/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// Default to localhost, but allow overriding via ENV variables
	storageNodes = []string{
		getEnv("STORAGE_NODE_1", "localhost:9001"),
		getEnv("STORAGE_NODE_2", "localhost:9002"),
		getEnv("STORAGE_NODE_3", "localhost:9003"),
	}
	clients []pb.StorageServiceClient
	enc     *encoder.Encoder

	// NEW: A simple memory map to store file sizes
	// In a real app, this would be a Postgres/Redis Database
	fileMetadata = make(map[string]int64)
)

func main() {
	// 1. Init Encoder
	var err error
	enc, err = encoder.NewEncoder()
	if err != nil {
		log.Fatalf("Failed to create encoder: %v", err)
	}

	// 2. Connect to Storage Nodes
	for _, addr := range storageNodes {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect to %s: %v", addr, err)
		}
		client := pb.NewStorageServiceClient(conn)
		clients = append(clients, client)
		fmt.Printf("✅ Connected to Storage Node at %s\n", addr)
	}

	// 3. Start HTTP Server
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Define Endpoints
	r.POST("/upload", handleUpload)
	r.GET("/download/:filename", handleDownload) // NEW Endpoint
	r.GET("/status", handleStatus)

	fmt.Println("🚀 Gateway running on http://localhost:8080")
	r.Run(":8080")
}

// ---------------- HANDLERS ----------------

func handleUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// NEW: Save the file size to our metadata map
	fileMetadata[fileHeader.Filename] = int64(len(data))

	shards, err := enc.Encode(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Encoding failed"})
		return
	}

	for i, shard := range shards {
		nodeIndex := i % len(clients)
		client := clients[nodeIndex]
		shardID := fmt.Sprintf("%s_shard_%d", fileHeader.Filename, i)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.StoreChunk(ctx, &pb.ChunkRequest{
			Id:   shardID,
			Data: shard,
		})

		if err != nil {
			fmt.Printf("❌ Failed to send shard %d to node %d: %v\n", i, nodeIndex, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded successfully",
		"filename": fileHeader.Filename,
		"size":     len(data),
	})
}

// NEW: The Download Logic
func handleDownload(c *gin.Context) {
	filename := c.Param("filename")

	// 1. Check if we know this file
	fileSize, exists := fileMetadata[filename]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found in metadata"})
		return
	}

	// 2. Fetch Shards from Storage Nodes (Parallel Fetch)
	// We need to reconstruct 6 shards (4 Data + 2 Parity)
	shards := make([][]byte, 6)
	var wg sync.WaitGroup

	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(shardIndex int) {
			defer wg.Done()

			// Identify which node has this shard
			nodeIndex := shardIndex % len(clients)
			client := clients[nodeIndex]
			shardID := fmt.Sprintf("%s_shard_%d", filename, shardIndex)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Call gRPC GetChunk
			resp, err := client.GetChunk(ctx, &pb.ChunkID{Id: shardID})

			if err == nil && resp.Found {
				shards[shardIndex] = resp.Data
				fmt.Printf("✅ Retrieved Shard %d from Node %d\n", shardIndex, nodeIndex)
			} else {
				fmt.Printf("⚠️ Failed to retrieve Shard %d (Node might be down)\n", shardIndex)
				shards[shardIndex] = nil // Mark as missing
			}
		}(i)
	}
	wg.Wait()

	// 3. Verify & Reconstruct (The Magic)
	// Even if some shards are nil, Reconstruct will try to fix them using the parity math
	err := enc.Reconstruct(shards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "File corrupted. Too many nodes down.", "details": err.Error()})
		return
	}

	// 4. Join shards back into original file
	// We set Content-Disposition so the browser downloads it instead of displaying binary
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")

	// Write the reconstructed data directly to the HTTP response
	err = enc.Join(c.Writer, shards, int(fileSize))
	if err != nil {
		fmt.Printf("Join failed: %v\n", err)
	}
}

func handleStatus(c *gin.Context) {
    type NodeStatus struct {
        ID     int    `json:"id"`
        Name   string `json:"name"`
        Status string `json:"status"` // "alive" or "dead"
    }

    var statuses []NodeStatus

    for i, addr := range storageNodes {
        status := "alive"
        
        // Try to connect to the TCP port directly
        // Timeout = 200ms
        conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
        if err != nil {
            status = "dead"
        } else {
            conn.Close()
        }

        statuses = append(statuses, NodeStatus{
            ID:     i + 1,
            Name:   fmt.Sprintf("Storage-%d", i+1),
            Status: status,
        })
    }

    c.JSON(http.StatusOK, statuses)
}

// Add this helper function at the bottom of the file
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
