package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	pb "github.com/imdinnesh/s3-go/internal/transport"
	"google.golang.org/grpc"
)

// server controls the storage logic
type server struct {
	pb.UnimplementedStorageServiceServer
	storageDir string
}

// StoreChunk implements the gRPC method we defined in the .proto file
func (s *server) StoreChunk(ctx context.Context, req *pb.ChunkRequest) (*pb.StoreReply, error) {
	fmt.Printf("📥 Received chunk: %s (%d bytes)\n", req.Id, len(req.Data))

	// Ensure storage directory exists
	filename := filepath.Join(s.storageDir, req.Id)
	
	// Write the bytes to disk
	err := ioutil.WriteFile(filename, req.Data, 0644)
	if err != nil {
		return &pb.StoreReply{Success: false, Message: err.Error()}, err
	}

	fmt.Printf("✅ Saved to %s\n", filename)
	return &pb.StoreReply{Success: true, Message: "Stored successfully"}, nil
}

// GetChunk implements the retrieval logic
func (s *server) GetChunk(ctx context.Context, req *pb.ChunkID) (*pb.ChunkData, error) {
	filename := filepath.Join(s.storageDir, req.Id)
	
	data, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return &pb.ChunkData{Found: false}, nil
	} else if err != nil {
		return nil, err
	}

	return &pb.ChunkData{Found: true, Data: data}, nil
}

func main() {
	// 1. Setup the Storage Folder (where files will live)
	// We use the port number to create distinct folders (e.g., /tmp/storage_9000)
	port := "9000" 
	if len(os.Args) > 1 {
		port = os.Args[1] // Allow passing port as argument
	}

	storageDir := fmt.Sprintf("storage_%s", port)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage dir: %v", err)
	}

	// 2. Start the TCP Listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 3. Start the gRPC Server
	s := grpc.NewServer()
	pb.RegisterStorageServiceServer(s, &server{storageDir: storageDir})
	
	fmt.Printf("🚀 Storage Node running on port %s. Saving to /%s\n", port, storageDir)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}