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

func (e *Engine) Initialize(block *types.Block) error {
	e.prevBlock = block
	return nil
}

func (e *Engine) StartBlockProduction(ctx context.Context) {
	ticker := time.NewTicker(e.blockRate)

	logrus.WithField("rate", e.blockRate).Info("starting block producer")
	if e.stopHeight > 0 {
		logrus.WithField("stop_height", e.stopHeight).Info("block production will stop at height")
		if e.prevBlock != nil && e.prevBlock.Height >= e.stopHeight {
			ticker.Stop()
		}
	}

	for {
		select {
		case <-ticker.C:
			block := e.createBlock()
			e.blockChan <- &block

			if e.stopHeight > 0 && block.Height >= e.stopHeight {
				logrus.Info("reached stop height")
				ticker.Stop()
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

func (e *Engine) createBlock() types.Block {
	block := types.Block{
		Timestamp:    time.Now().UTC(),
		Transactions: []types.Transaction{},
	}

	if e.prevBlock != nil { // Continue the chain
		block.Height = e.prevBlock.Height + 1
		block.Hash = makeHash(block.Height)
		block.PrevHash = e.prevBlock.Hash
	} else { // Start from genesis height
		logrus.WithField("height", e.genesisHeight).Info("starting from genesis block height")

		block.Height = e.genesisHeight
		block.Hash = makeHash(e.genesisHeight)
		block.PrevHash = makeHash(e.genesisHeight - 1)
	}

	for i := uint64(0); i < block.Height%10; i++ {
		tx := types.Transaction{
			Type:     "transfer",
			Hash:     makeHash(fmt.Sprintf("%v-%v", block.Height, i)),
			Sender:   "0xDEADBEEF",
			Receiver: "0xBAAAAAAD",
			Amount:   big.NewInt(int64(i * 1000000000)),
			Fee:      big.NewInt(10000),
			Success:  true,
			Events:   e.generateEvents(block.Height),
		}

		block.Transactions = append(block.Transactions, tx)
	}

	e.prevBlock = &block
	return block
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
