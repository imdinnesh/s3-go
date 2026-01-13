package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/imdinnesh/s3-go/internal/encoder"
	pb "github.com/imdinnesh/s3-go/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// List of our Storage Nodes (we will run 3 of them)
var storageNodes = []string{
	"localhost:9001",
	"localhost:9002",
	"localhost:9003",
}

func main() {
	// 1. Initialize the Reed-Solomon Encoder
	enc, err := encoder.NewEncoder() // Make sure your function name matches Day 1 code (NewTitanEncoder or NewEncoder)
	if err != nil {
		log.Fatalf("Failed to create encoder: %v", err)
	}

	// 2. Connect to ALL Storage Nodes
	var clients []pb.StorageServiceClient
	for _, addr := range storageNodes {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect to %s: %v", addr, err)
		}
		defer conn.Close()
		client := pb.NewStorageServiceClient(conn)
		clients = append(clients, client)
		fmt.Printf("✅ Connected to Storage Node at %s\n", addr)
	}

	// 3. Simulate an Upload
	filename := "secret_plans.txt"
	data := []byte("This is a super secret file that will be split into shards and distributed across the cluster!")
	
	fmt.Printf("\n📤 Uploading %s (%d bytes)...\n", filename, len(data))
	upload(enc, clients, filename, data)
}

func upload(enc *encoder.Encoder, clients []pb.StorageServiceClient, filename string, data []byte) {
	// A. Split and Encode the data (returns 6 shards: 4 Data + 2 Parity)
	shards, err := enc.Encode(data)
	if err != nil {
		log.Fatalf("Encoding failed: %v", err)
	}
	fmt.Printf("🪓 File split into %d shards.\n", len(shards))

	// B. Distribute shards via Round-Robin
	// Shard 0 -> Node 0
	// Shard 1 -> Node 1
	// Shard 2 -> Node 2
	// Shard 3 -> Node 0 ... etc
	for i, shard := range shards {
		nodeIndex := i % len(clients)
		client := clients[nodeIndex]

		// Create a unique ID for this shard: "filename_shardIndex"
		shardID := fmt.Sprintf("%s_shard_%d", filename, i)

		// Send via gRPC
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := client.StoreChunk(ctx, &pb.ChunkRequest{
			Id:   shardID,
			Data: shard,
		})

		if err != nil {
			fmt.Printf("❌ Failed to send Shard %d to Node %d: %v\n", i, nodeIndex, err)
		} else {
			fmt.Printf("🚀 Sent Shard %d (%d bytes) -> Node %d\n", i, len(shard), nodeIndex)
		}
	}
	fmt.Println("\n✅ Upload Complete!")
}