package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/streamingfast/dummy-blockchain/types"
)

type engineEvent struct {
	isBlock bool
	height  uint64
	index   int32
}

// TestStartBlockProduction_FlashPartialsOrderedBeforeBlock exercises the block/flash
// ticker at a fast rate and asserts every regular block (except the very first one,
// which has no prior flash cycle to draw from) is preceded by flash partials 1, 2, 3
// in order. This is a regression test for a race where two independent tickers firing
// on the same instant could let `select` pick the flash tick over the block tick,
// dropping a partial.
func TestStartBlockProduction_FlashPartialsOrderedBeforeBlock(t *testing.T) {
	engine := NewEngine("genesis", 1, time.Now(), 0, 6, 600, 64, false, false)
	require.NoError(t, engine.Initialize(nil, types.GenesisBlock("genesis", 1, time.Now())))

	blocks := engine.SubscribeBlocks()
	flashBlocks := engine.SubscribeFlashBlocks()

	go engine.StartBlockProduction(t.Context(), false, true)

	var events []engineEvent
	timeout := time.After(5 * time.Second)

	for blocks != nil || flashBlocks != nil {
		select {
		case b, ok := <-blocks:
			if !ok {
				blocks = nil
				continue
			}
			events = append(events, engineEvent{isBlock: true, height: b.Header.Height})

		case f, ok := <-flashBlocks:
			if !ok {
				flashBlocks = nil
				continue
			}
			events = append(events, engineEvent{height: f.Header.Height, index: f.Index})

		case <-timeout:
			require.Fail(t, "timed out waiting for engine to stop")
		}
	}

	checked := 0
	for i, ev := range events {
		if !ev.isBlock || ev.height <= 2 { // skip genesis and the first regular block: no prior flash cycle exists for either
			continue
		}

		// slots 1-3 send partials 1, 2, 3 for this height; slot 0 then sends the final
		// flash block (index 1004) immediately before the block itself.
		require.GreaterOrEqualf(t, i, 4, "block %d is missing its preceding partials", ev.height)

		for j := i - 4; j < i; j++ {
			require.Falsef(t, events[j].isBlock, "expected a flash partial before block %d", ev.height)
			require.Equalf(t, ev.height, events[j].height, "flash partial height mismatch before block %d", ev.height)
		}

		got := []int32{events[i-4].index, events[i-3].index, events[i-2].index, events[i-1].index}
		require.Equalf(t, []int32{1, 2, 3, 1004}, got, "block %d partial order", ev.height)
		checked++
	}

	require.GreaterOrEqual(t, checked, 3, "expected multiple blocks with ordered partials to have been observed")
}
