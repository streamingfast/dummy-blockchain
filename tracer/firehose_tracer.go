package tracer

import (
	"encoding/base64"
	"fmt"

	pbacme "github.com/streamingfast/dummy-blockchain/pb/sf/acme/type/v1"
	"github.com/streamingfast/dummy-blockchain/types"
	"google.golang.org/protobuf/proto"
)

var _ Tracer = &FirehoseTracer{}

type FirehoseTracer struct {
	activeBlock *pbacme.Block
	activeTrx   *pbacme.Transaction
}

// Initialize implements Tracer.
func (*FirehoseTracer) Initialize() error {
	fmt.Printf("FIRE INIT 1.0 %s\n", new(pbacme.Block).ProtoReflect().Descriptor().FullName())
	return nil
}

// OnBlockEnd implements Tracer.
func (t *FirehoseTracer) OnBlockEnd(blk *types.Block, finalBlockHeader *types.BlockHeader) {
	if t.activeBlock == nil {
		panic(fmt.Errorf("no active block, something is wrong in the tracer call order"))
	}

	header := t.activeBlock.Header

	previousNum := uint64(0)
	if header.PreviousNum != nil {
		previousNum = *header.PreviousNum
	}

	previousHash := ""
	if header.PreviousHash != nil {
		previousHash = *header.PreviousHash
	}

	blockPayload, err := proto.Marshal(t.activeBlock)
	if err != nil {
		panic(fmt.Errorf("unable to marshal block: %w", err))
	}

	fmt.Printf("FIRE BLOCK %d %s %d %s %d %d %s\n",
		header.Height,
		header.Hash,
		previousNum,
		previousHash,
		header.FinalNum,
		header.Timestamp,
		base64.StdEncoding.EncodeToString(blockPayload),
	)

	t.activeBlock = nil
	t.activeTrx = nil
}

// OnBlockStart implements Tracer.
func (t *FirehoseTracer) OnBlockStart(header *types.BlockHeader) {
	if t.activeBlock != nil {
		panic(fmt.Errorf("block already started, something is wrong in the tracer call order"))
	}

	t.activeBlock = &pbacme.Block{
		Header: &pbacme.BlockHeader{
			Height:    header.Height,
			Hash:      header.Hash,
			FinalNum:  header.FinalNum,
			FinalHash: header.FinalHash,
			Timestamp: header.Timestamp.UnixNano(),
		},
	}

	if header.PrevHash != nil {
		t.activeBlock.Header.PreviousNum = header.PrevNum
		t.activeBlock.Header.PreviousHash = header.PrevHash
	}
}

// OnTrxStart implements Tracer.
func (t *FirehoseTracer) OnTrxStart(trx *types.Transaction) {
	if t.activeTrx != nil {
		panic(fmt.Errorf("transaction already started, something is wrong in the tracer call order"))
	}

	t.activeTrx = &pbacme.Transaction{
		Type:     trx.Type,
		Hash:     trx.Hash,
		Sender:   trx.Sender,
		Receiver: trx.Receiver,
		Amount:   &pbacme.BigInt{Bytes: trx.Amount.Bytes()},
		Fee:      &pbacme.BigInt{Bytes: trx.Fee.Bytes()},
	}
}

// OnTrxEvent implements Tracer.
func (t *FirehoseTracer) OnTrxEvent(trxHash string, event *types.Event) {
	if t.activeTrx == nil {
		panic(fmt.Errorf("no active transaction, something is wrong in the tracer call order"))
	}

	pbEvent := &pbacme.Event{
		Type: event.Type,
	}

	if len(event.Attributes) > 0 {
		pbEvent.Attributes = make([]*pbacme.Attribute, len(event.Attributes))
		for i, attr := range event.Attributes {
			pbEvent.Attributes[i] = &pbacme.Attribute{
				Key:   attr.Key,
				Value: attr.Value,
			}
		}
	}

	t.activeTrx.Events = append(t.activeTrx.Events, pbEvent)
}

// OnTrxEnd implements Tracer.
func (t *FirehoseTracer) OnTrxEnd(trx *types.Transaction) {
	if t.activeTrx == nil {
		panic(fmt.Errorf("no active transaction, something is wrong in the tracer call order"))
	}

	t.activeTrx.Success = trx.Success

	t.activeBlock.Transactions = append(t.activeBlock.Transactions, t.activeTrx)
	t.activeTrx = nil
}
