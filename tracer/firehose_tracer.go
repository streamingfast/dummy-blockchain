package tracer

import (
	"encoding/base64"
	"fmt"

	"github.com/sirupsen/logrus"
	pbacme "github.com/streamingfast/dummy-blockchain/pb/sf/acme/type/v1"
	"github.com/streamingfast/dummy-blockchain/types"
	"google.golang.org/protobuf/proto"
)

var _ Tracer = &FirehoseTracer{}

type FirehoseTracer struct {
	activeBlock           *pbacme.Block
	activeTrx             *pbacme.Transaction
	withFlashBlocks       bool
	activeBlockFlashIndex int32
}

// Initialize implements Tracer.
func (t *FirehoseTracer) Initialize(version string) error {
	if version == "3.1" {
		t.withFlashBlocks = true
	}
	fmt.Printf("FIRE INIT %s %s\n", version, new(pbacme.Block).ProtoReflect().Descriptor().FullName())
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

	logrus.WithField("proto_size", len(blockPayload)).Debug("marshalled block to proto")

	t.printBlock(header, previousNum, previousHash, base64.StdEncoding.EncodeToString(blockPayload), 0)

	t.activeBlock = nil
	t.activeTrx = nil
}

func (t *FirehoseTracer) printBlock(header *pbacme.BlockHeader, prevNum uint64, prevHash string, blockPayload string, flashBlockIndex int32) {
	if t.withFlashBlocks {
		fmt.Printf("FIRE BLOCK %d %d %s %d %s %d %d %s\n",
			header.Height,
			flashBlockIndex,
			header.Hash,
			prevNum,
			prevHash,
			header.FinalNum,
			header.Timestamp,
			blockPayload,
		)
	} else {
		fmt.Printf("FIRE BLOCK %d %s %d %s %d %d %s\n",
			header.Height,
			header.Hash,
			prevNum,
			prevHash,
			header.FinalNum,
			header.Timestamp,
			blockPayload,
		)
	}
}

// OnFlashBlockEnd implements Tracer.
func (t *FirehoseTracer) OnFlashBlockEnd(blk *types.Block, finalBlockHeader *types.BlockHeader, flashBlockIndex int32) {
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
	t.printBlock(header, previousNum, previousHash, base64.StdEncoding.EncodeToString(blockPayload), flashBlockIndex)

	t.activeBlock = nil
	t.activeTrx = nil
}

// OnFlashBlockStart implements Tracer.
func (t *FirehoseTracer) OnFlashBlockStart(header *types.BlockHeader) {
	t.OnBlockStart(header)
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

	if header.PrevHash != nil && header.PrevNum != nil {
		t.activeBlock.Header.PreviousNum = header.PrevNum
		t.activeBlock.Header.PreviousHash = header.PrevHash
	}
}

func (t *FirehoseTracer) OnCommitmentSignal(sig *types.Signal) {
	fmt.Printf("FIRE SIGNAL 1 %d %s %d\n",
		sig.BlockNumber,
		sig.BlockID,
		sig.CommitmentLevel,
	)
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
		Data:     trx.Data,
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
