package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imdinnesh/s3-go/internal/encoder"
	pb "github.com/imdinnesh/s3-go/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	storageNodes = []string{
		"localhost:9001",
		"localhost:9002",
		"localhost:9003",
	}
	clients []pb.StorageServiceClient
	enc     *encoder.Encoder
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
		// Note: In a real app, we handle connection closing gracefully. 
		// For now, we let them stay open.
		client := pb.NewStorageServiceClient(conn)
		clients = append(clients, client)
		fmt.Printf("✅ Connected to Storage Node at %s\n", addr)
	}

	// 3. Start HTTP Server
	r := gin.Default()
	
	// Define the Upload Endpoint
	r.POST("/upload", handleUpload)

	fmt.Println("🚀 Gateway running on http://localhost:8080")
	r.Run(":8080")
}

func handleUpload(c *gin.Context) {
	// A. Parse the Multipart Form (File Upload)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// B. Open the file
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// C. Read file content (For now, we read into RAM. Week 2 we fix this!)
	data, err := ioutil.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// D. Shard and Distribute
	filename := fileHeader.Filename
	shards, err := enc.Encode(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Encoding failed"})
		return
	}

	// E. Send to Storage Nodes (Round Robin)
	for i, shard := range shards {
		nodeIndex := i % len(clients)
		client := clients[nodeIndex]
		shardID := fmt.Sprintf("%s_shard_%d", filename, i)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.StoreChunk(ctx, &pb.ChunkRequest{
			Id:   shardID,
			Data: shard,
		})

		if err != nil {
			fmt.Printf("❌ Failed to send shard %d to node %d: %v\n", i, nodeIndex, err)
			// In a real app, we would handle this error (retry/fail)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded and sharded successfully",
		"filename": filename,
		"shards": len(shards),
	})
}