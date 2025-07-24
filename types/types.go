package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"
)

func MakeHash(data any) string {
	return MakeHashNonce(data, nil)
}

func MakeHashNonce(data any, nonce *uint64) string {
	content := fmt.Appendf(nil, "%v", data)
	if nonce != nil {
		content = binary.LittleEndian.AppendUint64(content, *nonce)
	}

	shaSum := sha256.Sum256(content)
	return fmt.Sprintf("%x", shaSum)
}

func GenesisBlock(hash string, height uint64, genesisTime time.Time) *Block {
	header := &BlockHeader{
		Height:    height,
		Hash:      hash,
		FinalNum:  height,
		FinalHash: hash,
		Timestamp: genesisTime,
	}

	return &Block{
		Header:       header,
		Transactions: nil,
	}
}

type BlockHeader struct {
	Height    uint64    `json:"height"`
	Hash      string    `json:"hash"`
	PrevNum   *uint64   `json:"prev_num"`
	PrevHash  *string   `json:"prev_hash"`
	FinalNum  uint64    `json:"final_num"`
	FinalHash string    `json:"final_hash"`
	Timestamp time.Time `json:"timestamp"`
}

type Block struct {
	Header       *BlockHeader  `json:"header"`
	Transactions []Transaction `json:"transactions"`
}

// ApproximatedSize computes an approximation of how big the block would be
// once converted into a Protobuf Block model
func (b *Block) ApproximatedSize() int {
	size := 0

	// BlockHeader size approximation
	if b.Header != nil {
		size += 8 // Height (uint64)
		size += protoSizeOfString(b.Header.Hash)
		if b.Header.PrevNum != nil {
			size += 8 // PrevNum (uint64)
		}
		if b.Header.PrevHash != nil {
			size += protoSizeOfString(*b.Header.PrevHash)
		}
		size += 8 // FinalNum (uint64)
		size += protoSizeOfString(b.Header.FinalHash)
		size += 8 // Timestamp (int64 for protobuf)
	}

	// Transactions size approximation
	for _, tx := range b.Transactions {
		size += protoSizeOfString(tx.Type)
		size += protoSizeOfString(tx.Hash)
		size += protoSizeOfString(tx.Sender)
		size += protoSizeOfString(tx.Receiver)
		size += protoSizeOfBigInt(tx.Amount)
		size += protoSizeOfBigInt(tx.Fee)
		size += 1 // Success bool

		// Events size approximation
		for _, event := range tx.Events {
			size += protoSizeOfString(event.Type)

			// Attributes size approximation
			for _, attr := range event.Attributes {
				size += protoSizeOfString(attr.Key)
				size += protoSizeOfString(attr.Value)
			}
		}
	}

	// Add protobuf field tag overhead (1 byte per field approximately)
	numFields := 0
	if b.Header != nil {
		numFields += 7 // header fields
	}
	numFields += len(b.Transactions) * 8 // transaction fields
	for _, tx := range b.Transactions {
		numFields += len(tx.Events) * 2 // event fields
		for _, event := range tx.Events {
			numFields += len(event.Attributes) * 2 // attribute fields
		}
	}
	size += numFields

	return size
}

type Transaction struct {
	Type     string   `json:"type"`
	Hash     string   `json:"hash"`
	Sender   string   `json:"sender"`
	Receiver string   `json:"receiver"`
	Amount   *big.Int `json:"amount"`
	Fee      *big.Int `json:"fee"`
	Success  bool     `json:"success"`
	Events   []Event  `json:"events"`
}

type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// protoSizeOfString calculates the protobuf size of a string field
func protoSizeOfString(s string) int {
	if len(s) == 0 {
		return 0
	}
	// String length + varint encoding of length
	return len(s) + protoSizeOfVarint(len(s))
}

// protoSizeOfBytes calculates the protobuf size of a bytes field
func protoSizeOfBytes(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	// Bytes length + varint encoding of length
	return len(b) + protoSizeOfVarint(len(b))
}

// protoSizeOfBigInt calculates the protobuf size of a big.Int as bytes
func protoSizeOfBigInt(bi *big.Int) int {
	if bi == nil {
		return 0
	}
	return protoSizeOfBytes(bi.Bytes())
}

// protoSizeOfVarint calculates the size of a varint encoding
func protoSizeOfVarint(x int) int {
	if x < 0 {
		return 10 // negative numbers take 10 bytes
	}
	if x < (1 << 7) {
		return 1
	}
	if x < (1 << 14) {
		return 2
	}
	if x < (1 << 21) {
		return 3
	}
	if x < (1 << 28) {
		return 4
	}
	return 5
}
