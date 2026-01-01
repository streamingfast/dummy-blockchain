package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/bits"
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

// MakeFakeHash creates a fake 32-byte hash using bit operations from block height and tx index.
// This is for speed concerns, avoiding expensive SHA256 computation.
func MakeFakeHash(blockHeight uint64, txIndex int) string {
	var hash [32]byte

	// Spread block height across first 16 bytes using bit operations
	binary.BigEndian.PutUint64(hash[0:8], blockHeight)
	binary.BigEndian.PutUint64(hash[8:16], blockHeight^0xAAAAAAAAAAAAAAAA)

	// Spread tx index across next 16 bytes using bit operations
	txIndexU64 := uint64(txIndex)
	binary.BigEndian.PutUint64(hash[16:24], txIndexU64)
	binary.BigEndian.PutUint64(hash[24:32], (txIndexU64<<32)|(blockHeight&0xFFFFFFFF))

	return fmt.Sprintf("%x", hash)
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

type FlashBlock struct {
	*Block
	Index int32
}

type Signal struct {
	BlockID         string
	BlockNumber     uint64
	CommitmentLevel int32
}

// ApproximatedSize computes an approximation of how big the block would be
// once converted into a Protobuf Block model
func (b *Block) ApproximatedSize() int {
	size := 0

	// BlockHeader size approximation
	if b.Header != nil {
		size += protoSizeOfVarint(int(b.Header.Height)) // Height (uint64)
		size += protoSizeOfString(b.Header.Hash)
		if b.Header.PrevNum != nil {
			size += protoSizeOfVarint(int(*b.Header.PrevNum)) // PrevNum (uint64)
		}
		if b.Header.PrevHash != nil {
			size += protoSizeOfString(*b.Header.PrevHash)
		}
		size += protoSizeOfVarint(int(b.Header.FinalNum)) // FinalNum (uint64)
		size += protoSizeOfString(b.Header.FinalHash)
		size += protoSizeOfVarint(int(b.Header.Timestamp.UnixNano())) // Timestamp (int64 for protobuf)
	}

	// Transactions size approximation
	for _, tx := range b.Transactions {
		size += protoSizeOfString(tx.Type)
		size += protoSizeOfString(tx.Hash)
		size += protoSizeOfString(tx.Sender)
		size += protoSizeOfString(tx.Receiver)
		size += protoSizeOfBytes(tx.Data)
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
	Data     []byte   `json:"data,omitempty"`
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

const (
	_BYTE_COUNT_PER_WORD = _BIT_COUNT_PER_WORD / 8 // word size in bytes
	_BIT_COUNT_PER_WORD  = bits.UintSize           // word size in bits
)

// protoSizeOfBigInt calculates the protobuf size of a big.Int as bytes
func protoSizeOfBigInt(bi *big.Int) int {
	if bi == nil {
		return 0
	}

	byteCount := bi.BitLen() * _BYTE_COUNT_PER_WORD
	if byteCount == 0 {
		return 0
	}

	return byteCount + protoSizeOfVarint(byteCount)
}

// protoSizeOfVarint calculates the size of a varint encoding of an integer
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
