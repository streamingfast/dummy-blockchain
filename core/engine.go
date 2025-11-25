package core

import (
	"context"
	"fmt"
	"math/big"
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

func (e *Engine) StartBlockProduction(ctx context.Context, withSignal, withFlashBlocks bool) {
	ticker := time.NewTicker(e.blockRate)

	logrus.WithField("rate", e.blockRate).Info("starting block producer")
	if e.stopHeight > 0 {
		logrus.WithField("stop_height", e.stopHeight).Info("block production will stop at height")
		if e.prevBlock != nil && e.prevBlock.Header.Height >= e.stopHeight {
			ticker.Stop()
		}
	}

	var lastBlock *types.Block
	var lastSignal *types.Signal
	var lastFlashBlockNum uint64
	var lastFlashBlockIndex uint64

	signalTicker := time.NewTicker(e.blockRate)
	if withSignal {
		<-time.After(e.blockRate / 2) // offset by half duration
		signalTicker.Reset(e.blockRate)
	} else {
		signalTicker.Stop()
	}

	flashRate := e.blockRate / 5 // we use 4 slots out of 5
	flashBlockTicker := time.NewTicker(flashRate)
	if !withFlashBlocks {
		flashBlockTicker.Stop()
	}

	for {
		select {
		case <-ticker.C:
			for _, block := range e.createBlocks() {
				e.blockChan <- block
				lastBlock = block

				if e.stopHeight > 0 && block.Header.Height >= e.stopHeight {
					logrus.Info("reached stop height")
					ticker.Stop()
					return
				}
			}
		case <-signalTicker.C:
			if !withSignal {
				continue // just ignore if a signal ticker comes in, but it actually should not be called because of the Stop(), unless there is a crazy race condtition
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
				continue // just ignore if a flashblock ticker comes in, but it actually should not be called because of the Stop(), unless there is a crazy race condtition
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
			logrus.Info("stopping block producer")
			close(e.blockChan)
			return
		}
	}
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
	if e.prevBlock == nil {
		genesisBlock := types.GenesisBlock(e.genesisHash, e.genesisHeight, e.genesisTime)
		logrus.WithField("block", blockRef{genesisBlock.Header.Hash, e.genesisHeight}).Info("starting from genesis block height")
		e.prevBlock = genesisBlock
		e.finalBlock = genesisBlock

		out = append(out, genesisBlock)

		for len(out)-1 < int(e.genesisBlockBurst) {
			out = append(out, e.createBlocks()...)
		}

		return
	}

	heightToProduce := e.prevBlock.Header.Height + 1
	if e.withSkippedBlocks && heightToProduce%13 == 0 {
		heightToProduce += 1
		logrus.Info(fmt.Sprintf("skipping block #%d that is a multiple of 13, producing %d instead", heightToProduce-1, heightToProduce))
	}

	if e.withReorgs && heightToProduce%17 == 0 {
		if heightToProduce%2 == 0 {
			logrus.Info("producing 2 block fork sequence")
			firstFork := e.newBlock(heightToProduce, ptr(uint64(1)), e.prevBlock)
			secondFork := e.newBlock(heightToProduce+1, ptr(uint64(2)), firstFork)

			out = append(out, firstFork, secondFork)
		} else {
			logrus.Info("producing 1 block fork sequence")
			out = append(out, e.newBlock(heightToProduce, ptr(uint64(1)), e.prevBlock))
		}
	}

	block := e.newBlock(heightToProduce, nil, e.prevBlock)

	e.addTransactions(block, e.blockSizeInBytes)

	out = append(out, block)

	e.prevBlock = block
	if block.Header.Height%10 == 0 {
		logrus.WithField("block", blockRef{block.Header.Hash, block.Header.Height}).Info("produced block is now the final block")
		e.finalBlock = block
	}

	return
}

func (e *Engine) addTransactions(block *types.Block, sizeInBytes int) {

	for size := block.ApproximatedSize(); size < sizeInBytes; size = block.ApproximatedSize() {
		i := len(block.Transactions)

		block.Transactions = append(block.Transactions, types.Transaction{
			Type:     "transfer",
			Hash:     types.MakeHash(fmt.Sprintf("%v-%v", block.Header.Height, i)),
			Sender:   "0xDEADBEEF",
			Receiver: "0xBAAAAAAD",
			Amount:   big.NewInt(int64(i * 1000000000)),
			Fee:      big.NewInt(10000),
			Success:  true,
			Events:   e.generateEvents(block.Header.Height),
		})
	}
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

	switch {
	case height%3 == 0:
		events = append(events, types.Event{
			Type: "token_transfer",
			Attributes: []types.Attribute{
				{Key: "foo", Value: "bar"},
			},
		})
	case height%5 == 0:
		events = append(events, types.Event{
			Type: "delegate",
			Attributes: []types.Attribute{
				{Key: "delegator", Value: "addr1"},
				{Key: "validator", Value: "addr2"},
				{Key: "amount", Value: "123456789"},
			},
		})
	case height%3 == 0 && height%5 == 0:
		events = append(events, types.Event{
			Type: "coin_spent",
			Attributes: []types.Attribute{
				{Key: "spender", Value: "fizz"},
				{Key: "amount", Value: "buzz"},
			},
		})
	}

	return events
}

func ptr[T any](t T) *T {
	return &t
}
