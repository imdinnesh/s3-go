package encoder

import (
	"fmt"
	"io"
	"github.com/klauspost/reedsolomon"
)

// We want 4 Data Shards + 2 Parity Shards = 6 Total Shards
const (
	DataShards   = 4
	ParityShards = 2
)

type Encoder struct {
	enc reedsolomon.Encoder
}

func NewEncoder() (*Encoder, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, err
	}
	return &Encoder{enc: enc}, nil
}

// Encode takes file bytes and returns a list of byte slices (shards)
func (t *Encoder) Encode(data []byte) ([][]byte, error) {
	// 1. Split the data into shards
	shards, err := t.enc.Split(data)
	if err != nil {
		return nil, fmt.Errorf("split failed: %w", err)
	}

	// 2. Calculate the Parity Shards (The Math happens here)
	if err := t.enc.Encode(shards); err != nil {
		return nil, fmt.Errorf("encode failed: %w", err)
	}

	// shards now contains 4 Data + 2 Parity
	return shards, nil
}

// Reconstruct takes a list of shards (some might be nil/missing) and fixes them
func (t *Encoder) Reconstruct(shards [][]byte) error {
	// This checks if we have enough shards to rebuild
	ok, _ := t.enc.Verify(shards)
	if ok {
		return nil // No reconstruction needed
	}
	
	// Magic: Rebuild missing shards
	if err := t.enc.Reconstruct(shards); err != nil {
		return fmt.Errorf("reconstruction failed: %w", err)
	}
	
	return nil
}

// Join merges shards back into the original file
func (t *Encoder) Join(dst io.Writer, shards [][]byte, outSize int) error {
    return t.enc.Join(dst, shards, outSize)
}