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
