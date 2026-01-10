package main

import (
	"bytes"
	"fmt"

	"github.com/imdinnesh/s3-go/internal/encoder"
)

func main() {
	// 1. The original data
	originalData := []byte("This is a distributed system that never loses data!")
	fmt.Printf("Original: %s\n", originalData)

	// 2. Initialize Encoder
	enc, _ := encoder.NewEncoder()

	// 3. Split & Encode
	shards, _ := enc.Encode(originalData)
	fmt.Printf("Encoded into %d shards.\n", len(shards))

	// 4. SIMULATE DISASTER: Delete 2 shards (Set to nil)
	// Let's delete a Data Shard (index 0) and a Parity Shard (index 5)
	fmt.Println("⚠️  Disaster! Deleting Shard 0 and Shard 5...")
	shards[0] = nil 
	shards[5] = nil

	// 5. Attempt Recovery
	fmt.Println("🚑 Attempting reconstruction...")
	err := enc.Reconstruct(shards)
	if err != nil {
		panic(err)
	}
	fmt.Println("✅ Reconstruction successful!")

	// 6. Verify Output
	var buf bytes.Buffer
	enc.Join(&buf, shards, len(originalData))
	fmt.Printf("Recovered: %s\n", buf.String())
}