package deepmind

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"

	"google.golang.org/protobuf/proto"

	pbcodec "github.com/streamingfast/dummy-blockchain/pb/sf/acme/codec/v1"
	"github.com/streamingfast/dummy-blockchain/types"
)

var (
	Enabled bool
	writer  io.WriteCloser
)

func Enable(w io.WriteCloser) {
	Enabled = true
	writer = w
}

func SetWriter(w io.WriteCloser) {
	writer = w
}

func Shutdown() {
	writer.Close()
}

// BeginBlock marks the beginning of the block data for a single height
func BeginBlock(number uint64) {
	fmt.Fprintf(writer, "DMLOG BLOCK_BEGIN %d\n", number)
}

// Block writes all block data
func Block(block *types.Block) {
	newBlock := &pbcodec.Block{
		Height:       block.Height,
		Hash:         block.Hash,
		PrevHash:     block.PrevHash,
		Transactions: make([]*pbcodec.Transaction, len(block.Transactions)),
		Timestamp:    uint64(block.Timestamp.UnixNano()),
	}

	for idx, tx := range block.Transactions {
		events := make([]*pbcodec.Event, len(tx.Events))

		for idxEv, ev := range tx.Events {
			events[idxEv] = &pbcodec.Event{
				Type: ev.Type,
			}

			for _, attr := range ev.Attributes {
				events[idxEv].Attributes = append(events[idxEv].Attributes, &pbcodec.Attribute{
					Key:   attr.Key,
					Value: attr.Value,
				})
			}
		}

		newBlock.Transactions[idx] = &pbcodec.Transaction{
			Type:     tx.Type,
			Hash:     tx.Hash,
			Sender:   tx.Sender,
			Receiver: tx.Receiver,
			Amount: &pbcodec.BigInt{
				Bytes: tx.Amount.Bytes(),
			},
			Fee: &pbcodec.BigInt{
				Bytes: tx.Fee.Bytes(),
			},
			Success: tx.Success,
			Events:  events,
		}
	}

	data, err := proto.Marshal(newBlock)
	if err != nil {
		// Terminating the app here will cause the chain to halt so it does not
		// advance to a new block.
		log.Fatal(err)
	}

	fmt.Fprintf(writer, "DMLOG BLOCK_DATA %s\n", base64.StdEncoding.EncodeToString(data))
}

// EndBlock marks the end of the block data for a single height
func EndBlock(number uint64) {
	fmt.Fprintf(writer, "DMLOG BLOCK_END %d\n", number)
}
