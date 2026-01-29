package core

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streamingfast/dummy-blockchain/types"
)

type Engine struct {
	genesisHash       string
	genesisHeight     uint64
	genesisTime       time.Time
	genesisBlockBurst uint64
	stopHeight        uint64
	blockSizeInBytes  int
	blockRate         time.Duration
	blockChan         chan *types.Block
	flashBlockChan    chan *types.FlashBlock
	signalChan        chan *types.Signal
	prevBlock         *types.Block
	finalBlock        *types.Block
	tearedDown        bool
	teardownOnce      sync.Once
	withSkippedBlocks bool
	withReorgs        bool
}

func NewEngine(genesisHash string, genesisHeight uint64, genesisTime time.Time, genesisBlockBurst uint64, stopHeight uint64, rate int, blockSizeInBytes int, withSkippedBlocks bool, withReorgs bool) Engine {
	blockRate := time.Minute / time.Duration(rate)

	return Engine{
		genesisHash:       genesisHash,
		genesisHeight:     genesisHeight,
		genesisTime:       genesisTime,
		genesisBlockBurst: genesisBlockBurst,
		stopHeight:        stopHeight,
		blockRate:         blockRate,
		blockSizeInBytes:  blockSizeInBytes,
		blockChan:         make(chan *types.Block),
		signalChan:        make(chan *types.Signal),
		flashBlockChan:    make(chan *types.FlashBlock),
		tearedDown:        false,
		teardownOnce:      sync.Once{},
		withSkippedBlocks: withSkippedBlocks,
		withReorgs:        withReorgs,
	}
}

func (e *Engine) Initialize(prevBlock *types.Block, finalBlock *types.Block) error {
	e.prevBlock = prevBlock
	e.finalBlock = finalBlock

	if finalBlock == nil {
		return fmt.Errorf("final block cannot be nil")
	}

	return nil
}

func (e *Engine) stop(reason string, tickers ...*time.Ticker) {
	e.teardownOnce.Do(func() {
		logrus.Info(reason)

		e.tearedDown = true

		close(e.blockChan)
		close(e.signalChan)
		close(e.flashBlockChan)

		for _, ticker := range tickers {
			ticker.Stop()
		}
	})
}

func (e *Engine) StartBlockProduction(ctx context.Context, withCommitmentSignal, withFlashBlocks bool) {
	logrus.
		WithField("genesis_burst", e.genesisBlockBurst).
		WithField("rate", e.blockRate).
		WithField("size", e.blockSizeInBytes).
		WithField("stop_height", e.stopHeight).
		Info("starting block producer")

	if e.prevBlock == nil {
		genesisBlock := types.GenesisBlock(e.genesisHash, e.genesisHeight, e.genesisTime)
		logrus.WithField("block", blockRef{genesisBlock.Header.Hash, e.genesisHeight}).WithField("burst", e.genesisBlockBurst).Info("starting from genesis block height")
		e.prevBlock = genesisBlock
		e.finalBlock = genesisBlock

		e.blockChan <- genesisBlock

		startBurst := time.Now()
		for i := 0; i < int(e.genesisBlockBurst); {
			for _, block := range e.createBlocks() {
				if e.hasReachedStopHeight(block.Header.Height) {
					e.stop("reached stop block height during genesis burst")
					return
				}

				e.blockChan <- block
				i++
			}
		}
		logrus.WithField("duration", time.Since(startBurst).String()).Infof("genesis block burst of %d blocks produced", int(e.genesisBlockBurst))
	}

	var lastBlock *types.Block
	var lastSignal *types.Signal
	var lastFlashBlockNum uint64
	var lastFlashBlockIndex uint64

	blockTicker := time.NewTicker(e.blockRate)
	commitmentSignalTicker := time.NewTicker(e.blockRate)
	flashBlockTicker := time.NewTicker(e.blockRate / 5) // we use 4 slots out of 5

	if withCommitmentSignal {
		<-time.After(e.blockRate / 2) // offset by half duration
		commitmentSignalTicker.Reset(e.blockRate)
	} else {
		commitmentSignalTicker.Stop()
	}

	if !withFlashBlocks {
		flashBlockTicker.Stop()
	}

	for {
		if e.tearedDown {
			logrus.Info("block producer has been stopped")
			return
		}

		select {
		case <-blockTicker.C:
			for _, block := range e.createBlocks() {
				if e.hasReachedStopHeight(block.Header.Height) {
					e.stop("reached stop block height", blockTicker, commitmentSignalTicker, flashBlockTicker)
					return
				}

				e.blockChan <- block
				lastBlock = block
			}
		case <-commitmentSignalTicker.C:
			if !withCommitmentSignal {
				// Just ignore if a signal ticker comes in, but it actually should not be called because of the Stop(), unless there is a crazy race condition
				continue
			}
			if lastBlock != nil {
				if lastSignal != nil && lastSignal.BlockID == lastBlock.Header.Hash {
					continue // don't send duplicate signal
				}
				sig := &types.Signal{
					BlockID:         lastBlock.Header.Hash,
					BlockNumber:     lastBlock.Header.Height,
					CommitmentLevel: 10,
				}
				e.signalChan <- sig
				lastSignal = sig
			}

		case <-flashBlockTicker.C:
			if !withFlashBlocks {
				// Just ignore if a flashblock ticker comes in, but it actually should not be called because of the Stop(), unless there is a crazy race condition
				continue
			}
			if lastBlock == nil {
				continue
			}

			num := lastBlock.Header.Height + 1
			if num != lastFlashBlockNum {
				lastFlashBlockIndex = 0
				lastFlashBlockNum = num
				continue
			}

			idx := lastFlashBlockIndex + 1
			flashBlock := e.newBlock(num, &idx, e.prevBlock)
			e.addTransactions(flashBlock, int(idx*uint64(e.blockSizeInBytes)/4))
			e.flashBlockChan <- &types.FlashBlock{
				Block: flashBlock,
				Index: int32(idx),
			}

			lastFlashBlockIndex = idx
			lastFlashBlockNum = num

		case <-ctx.Done():
			e.stop("context done", blockTicker, commitmentSignalTicker, flashBlockTicker)
			return
		}
	}
}

func (e *Engine) hasReachedStopHeight(height uint64) bool {
	if e.stopHeight == 0 {
		return false
	}

	return height > e.stopHeight
}

func (e *Engine) SubscribeBlocks() <-chan *types.Block {
	return e.blockChan
}

func (e *Engine) SubscribeSignals() <-chan *types.Signal {
	return e.signalChan
}

func (e *Engine) SubscribeFlashBlocks() <-chan *types.FlashBlock {
	return e.flashBlockChan
}

func (e *Engine) createBlocks() (out []*types.Block) {
	heightToProduce := e.prevBlock.Header.Height + 1
	if e.withSkippedBlocks && heightToProduce%13 == 0 {
		heightToProduce += 1
		logrus.Info(fmt.Sprintf("skipping block #%d that is a multiple of 13, created %d instead", heightToProduce-1, heightToProduce))
	}

	if e.withReorgs && heightToProduce%17 == 0 {
		if heightToProduce%2 == 0 {
			logrus.Info("created 2 block fork sequence")
			firstFork := e.newBlock(heightToProduce, ptr(uint64(1)), e.prevBlock)
			secondFork := e.newBlock(heightToProduce+1, ptr(uint64(2)), firstFork)

			out = append(out, firstFork, secondFork)
		} else {
			logrus.Info("created 1 block fork sequence")
			out = append(out, e.newBlock(heightToProduce, ptr(uint64(1)), e.prevBlock))
		}
	}

	block := e.newBlock(heightToProduce, nil, e.prevBlock)
	e.addTransactions(block, e.blockSizeInBytes)

	out = append(out, block)

	e.prevBlock = block
	if block.Header.Height%10 == 0 {
		logrus.WithField("block", blockRef{block.Header.Hash, block.Header.Height}).Info("created block is now the final block")
		e.finalBlock = block
	}

	return
}

var simulateTypes = []string{"transfer", "delegate", "undelegate", "reward", "slash"}
var bigZero = big.NewInt(0)

const (
	KiB = 1024
)

func (e *Engine) addTransactions(block *types.Block, sizeInBytes int) {
	addTx := func(data []byte) {
		i := len(block.Transactions)

		txHash := types.MakeFakeHash(block.Header.Height, i)
		sender := "0x" + txHash[:40]
		receiver := "0x" + txHash[24:64]
		amount := new(big.Int).SetUint64((block.Header.Height << 32) | uint64(i))
		success := true

		// Each five transactions, make a fixed sender
		if i%7 == 0 {
			sender = "0xDEADBEEF"
		}

		// Each eleven transactions, make a fixed receiver
		if i%11 == 0 {
			receiver = "0xBAAAAAAD"
		}

		// Each 3 transactions, make amount zero
		if i%3 == 0 {
			amount = bigZero
		}

		// Each 13 transactions, make it fail
		if i%13 == 0 {
			success = false
		}

		block.Transactions = append(block.Transactions, types.Transaction{
			Type:     simulateTypes[i%len(simulateTypes)],
			Hash:     txHash,
			Sender:   sender,
			Receiver: receiver,
			Data:     fillData(data, block.Header.Height, i),
			Amount:   amount,
			Fee:      new(big.Int).SetUint64(block.Header.Height + uint64(i)),
			Success:  success,
			Events:   e.generateEvents(block.Header.Height),
		})
	}

	// At those small size, the loop below is fast enough
	if sizeInBytes < 10*KiB {
		for size := block.ApproximatedSize(); size <= sizeInBytes; size = block.ApproximatedSize() {
			addTx(make([]byte, 32))
		}
	}

	// At larger sizes, we estimate the number of transactions to add and generate data
	txCount := targetTxCount(sizeInBytes)
	dataSizePerTx := (sizeInBytes / txCount) - 250 // rough estimate of tx overhead

	// Ensure dataSizePerTx is non-negative to avoid makeslice panic
	if dataSizePerTx < 0 {
		dataSizePerTx = 0
	}

	for range txCount {
		addTx(make([]byte, dataSizePerTx))
	}
}

// targetTxCount provides an estimated target transaction count, and it gives roughly
// in that range:
//
//   - 100 transactions at 100KiB
//   - 1000 transactions at 1MiB
//   - 2500 transactions at 2.5MiB
//   - 5000 transactions at 5MiB
//   - 10000 transactions at 10MiB
//   - 100000 transactions at 100MiB
//   - etc.
//
// With a minimum of 10 transactions.
func targetTxCount(blockSizeInBytes int) int {
	scale := math.Log10(float64(blockSizeInBytes))
	exponent := scale - 3
	if exponent < 1 {
		exponent = 1
	}

	return int(math.Pow(10, float64(exponent)))
}

func fillData(buf []byte, blockHeight uint64, txIndex int) []byte {
	for i := range buf {
		buf[i] = byte((blockHeight + uint64(txIndex) + uint64(i)) % 256)
	}

	return buf
}

func (e *Engine) newBlock(height uint64, nonce *uint64, parent *types.Block) *types.Block {
	return &types.Block{
		Header: &types.BlockHeader{
			Height:    height,
			Hash:      types.MakeHashNonce(height, nonce),
			PrevNum:   &parent.Header.Height,
			PrevHash:  &parent.Header.Hash,
			FinalNum:  e.finalBlock.Header.Height,
			FinalHash: e.finalBlock.Header.Hash,
			Timestamp: e.genesisTime.Add(e.blockRate * time.Duration(height)),
		},
		Transactions: []types.Transaction{},
	}
}

func (e *Engine) generateEvents(height uint64) []types.Event {
	events := []types.Event{}

	events = append(events, types.Event{
		Type: "token_transfer",
		Attributes: []types.Attribute{
			{Key: "foo", Value: "bar"},
		},
	})

	switch {
	case height%2 == 0:
		events = append(events, types.Event{
			Type: "coin_spent",
			Attributes: []types.Attribute{
				{Key: "spender", Value: "fizz"},
				{Key: "amount", Value: "buzz"},
			},
		})
	case height%3 == 0:
		events = append(events, types.Event{
			Type: "delegate",
			Attributes: []types.Attribute{
				{Key: "delegator", Value: "addr1"},
				{Key: "validator", Value: "addr2"},
				{Key: "amount", Value: "123456789"},
			},
		})
	case height%5 == 0:
		events = append(events, types.Event{
			Type: "undelegate",
			Attributes: []types.Attribute{
				{Key: "delegator", Value: "addr1"},
				{Key: "amount", Value: "123456789"},
			},
		})
	}

	return events
}

func ptr[T any](t T) *T {
	return &t
}
