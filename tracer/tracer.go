package tracer

import (
	"github.com/streamingfast/dummy-blockchain/types"
)

type Tracer interface {
	Initialize(version string) error

	OnBlockStart(header *types.BlockHeader)

	OnFlashBlockStart(header *types.BlockHeader)

	OnCommitmentSignal(sig *types.Signal)

	OnTrxStart(trx *types.Transaction)

	OnTrxEvent(trxHash string, event *types.Event)

	OnTrxEnd(trx *types.Transaction)

	OnBlockEnd(blk *types.Block, finalBlockHeader *types.BlockHeader)

	OnFlashBlockEnd(blk *types.Block, finalBlockHeader *types.BlockHeader, idx int32)
}
