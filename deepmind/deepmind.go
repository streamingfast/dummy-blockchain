package deepmind

import (
	"encoding/hex"
	"fmt"
	"github.com/streamingfast/dummy-blockchain/types"
	"io"
	"math/big"
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
	// FIRE BLOCK_BEGIN <BLOCK_HEIGHT>
	fmt.Fprintf(writer, "FIRE BLOCK_BEGIN %d\n", number)
}

// BeginTrx marks the beginning of a transaction
func BeginTrx(trx *types.Transaction) {
	// FIRE BEGIN_TRX <HASH> <TYPE> <SENDER> <RECEIVER> <AMOUNT> <FEE> <SUCCESS>
	trxAmount := "0"
	if trx.Amount.Cmp(new(big.Int).SetUint64(0)) > 0 {
		trxAmount = hex.EncodeToString(trx.Amount.Bytes())
	}

	fmt.Fprintf(writer, "FIRE BEGIN_TRX %s %s %s %s %s %s %t\n",
		trx.Hash,
		trx.Type,
		trx.Sender,
		trx.Receiver,
		trxAmount,
		hex.EncodeToString(trx.Fee.Bytes()),
		trx.Success,
	)
}

// TrxBeginEvent records the beginning of an event
func TrxBeginEvent(trxHash string, event *types.Event) {
	// FIRE TRX_BEGIN_EVENT <TRX_HASH> <EVENT_TYPE>
	fmt.Fprintf(writer, "FIRE TRX_BEGIN_EVENT %s %s\n",
		trxHash,
		event.Type,
	)
}

// TrxEventAttr record an attribute for a given event
func TrxEventAttr(trxHash string, eventIndex uint64, key string, value string) {
	// FIRE TRX_BEGIN_EVENT <TRX_HASH> <EVENT_INDEX> <KEY> <VALUE>
	fmt.Fprintf(writer, "FIRE TRX_EVENT_ATTR %s %d %s %s\n",
		trxHash,
		eventIndex,
		key,
		value,
	)
}

// EndBlock marks the end of the block data for a single height
func EndBlock(blk *types.Block) {
	// FIRE BLOCK_END <NUMBER> <HASH> <PREV_HASH> <TIMESTAMP> <TRX_COUNT>
	fmt.Fprintf(writer, "FIRE BLOCK_END %d %s %s %d %d\n",
		blk.Height,
		blk.Hash,
		blk.PrevHash,
		blk.Timestamp.UnixNano(),
		len(blk.Transactions),
	)
}
