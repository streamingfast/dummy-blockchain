package tracer

import (
	"github.com/streamingfast/dummy-blockchain/types"
)

type Tracer interface {
	Initialize() error

	OnBlockStart(header *types.BlockHeader)

	OnTrxStart(trx *types.Transaction)

	OnTrxEvent(trxHash string, event *types.Event)

	OnTrxEnd(trx *types.Transaction)

	OnBlockEnd(blk *types.Block, finalBlockHeader *types.BlockHeader)
}
