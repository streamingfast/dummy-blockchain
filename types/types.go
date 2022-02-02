package types

import (
	"math/big"
	"time"
)

type Block struct {
	Height       uint64        `json:"height"`
	Hash         string        `json:"hash"`
	PrevHash     string        `json:"prev_hash"`
	Timestamp    time.Time     `json:"timestamp"`
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
