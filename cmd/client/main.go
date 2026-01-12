package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/imdinnesh/s3-go/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Connect to the Storage Node at localhost:9001
	conn, err := grpc.Dial("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewStorageServiceClient(conn)

	// 2. Send a Fake Chunk
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id := "test_file_shard_1"
	data := []byte("This is some binary data inside a shard.")

	r, err := c.StoreChunk(ctx, &pb.ChunkRequest{Id: id, Data: data})
	if err != nil {
		log.Fatalf("could not store: %v", err)
	}
	
	fmt.Printf("Server Response: %s (Success: %v)\n", r.Message, r.Success)
}