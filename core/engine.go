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
	genesisHeight uint64
	stopHeight    uint64
	blockRate     time.Duration
	blockChan     chan *types.Block
	prevBlock     *types.Block
	finalBlock    *types.Block
}

func NewEngine(genesisHeight, stopHeight uint64, rate int) Engine {
	blockRate := time.Minute / time.Duration(rate)

	if genesisHeight == 0 {
		genesisHeight = 1
	}

	return Engine{
		genesisHeight: genesisHeight,
		stopHeight:    stopHeight,
		blockRate:     blockRate,
		blockChan:     make(chan *types.Block),
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

func (e *Engine) StartBlockProduction(ctx context.Context) {
	ticker := time.NewTicker(e.blockRate)

	logrus.WithField("rate", e.blockRate).Info("starting block producer")
	if e.stopHeight > 0 {
		logrus.WithField("stop_height", e.stopHeight).Info("block production will stop at height")
		if e.prevBlock != nil && e.prevBlock.Header.Height >= e.stopHeight {
			ticker.Stop()
		}
	}

	for {
		select {
		case <-ticker.C:
			for _, block := range e.createBlocks() {
				e.blockChan <- block

				if e.stopHeight > 0 && block.Header.Height >= e.stopHeight {
					logrus.Info("reached stop height")
					ticker.Stop()
					return
				}
			}
		case <-ctx.Done():
			logrus.Info("stopping block producer")
			close(e.blockChan)
			return
		}
	}
}

func (e *Engine) Subscription() <-chan *types.Block {
	return e.blockChan
}

func (e *Engine) createBlocks() (out []*types.Block) {
	if e.prevBlock == nil {
		genesisBlock := types.GenesisBlock(e.genesisHeight)
		logrus.WithField("block", blockRef{genesisBlock.Header.Hash, e.genesisHeight}).Info("starting from genesis block height")
		e.prevBlock = genesisBlock
		e.finalBlock = genesisBlock

		out = append(out, genesisBlock)
		return
	}

	heightToProduce := e.prevBlock.Header.Height + 1
	if heightToProduce%13 == 0 {
		heightToProduce += 1
		logrus.Info(fmt.Sprintf("skipping block #%d that is a multiple of 13, producing %d instead", heightToProduce-1, heightToProduce))
	}

	if heightToProduce%17 == 0 {
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

	trxCount := min(heightToProduce%10, 500)
	for i := uint64(0); i < trxCount; i++ {
		tx := types.Transaction{
			Type:     "transfer",
			Hash:     types.MakeHash(fmt.Sprintf("%v-%v", heightToProduce, i)),
			Sender:   "0xDEADBEEF",
			Receiver: "0xBAAAAAAD",
			Amount:   big.NewInt(int64(i * 1000000000)),
			Fee:      big.NewInt(10000),
			Success:  true,
			Events:   e.generateEvents(heightToProduce),
		}

		block.Transactions = append(block.Transactions, tx)
	}

	out = append(out, block)

	e.prevBlock = block
	if block.Header.Height%10 == 0 {
		logrus.WithField("block", blockRef{block.Header.Hash, block.Header.Height}).Info("produced block is now the final block")
		e.finalBlock = block
	}

	return
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
