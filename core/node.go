package core

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/streamingfast/dummy-chain/deepmind"
	"github.com/streamingfast/dummy-chain/types"
)

type Node struct {
	engine Engine
	store  Store
}

func NewNode(storeDir string, blockRate int, genesisHeight uint64) *Node {
	return &Node{
		engine: NewEngine(genesisHeight, blockRate),
		store:  NewStore(storeDir),
	}
}

func (node *Node) Initialize() error {
	logrus.Info("initializing node")

	logrus.Info("initializing store")
	if err := node.store.Initialize(); err != nil {
		logrus.WithError(err).Error("store initialization failed")
		return err
	}

	var tipBlock *types.Block

	if tip := node.store.meta.TipHeight; tip > 0 {
		logrus.WithField("tip", tip).Info("loading last block")
		block, err := node.store.ReadBlock(tip)
		if err != nil {
			logrus.WithError(err).Error("cant read last block")
			return err
		}
		tipBlock = block
	}

	logrus.Info("initializing engine")
	if err := node.engine.Initialize(tipBlock); err != nil {
		logrus.WithError(err).Error("engine initialization failed")
		return err
	}

	return nil
}

func (node *Node) Start(ctx context.Context) error {
	go node.engine.StartBlockProduction(ctx)

	for {
		select {
		case block, ok := <-node.engine.Subscription():
			if !ok {
				return nil
			}
			if err := node.processBlock(block); err != nil {
				logrus.WithError(err).Error("failed to process block")
				return err
			}

			if deepmind.Enabled {
				deepmind.BeginBlock(block.Height)
				deepmind.Block(block)
				deepmind.EndBlock(block.Height)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (node *Node) processBlock(block *types.Block) error {
	logrus.
		WithField("height", block.Height).
		WithField("hash", block.Hash).
		Info("processing block")

	if err := node.store.WriteBlock(block); err != nil {
		return err
	}

	return nil
}
