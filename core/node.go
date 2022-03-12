package core

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/streamingfast/dummy-blockchain/deepmind"
	"github.com/streamingfast/dummy-blockchain/types"
)

type Node struct {
	engine Engine
	server Server
	store  *Store
}

func NewNode(storeDir string, blockRate int, genesisHeight uint64, serverAddr string) *Node {
	store := NewStore(storeDir)

	return &Node{
		engine: NewEngine(genesisHeight, blockRate),
		store:  store,
		server: NewServer(store, serverAddr),
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
	go node.server.Start() // TODO: handle error here
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
				for _, trx := range block.Transactions {
					deepmind.BeginTrx(&trx)
					for idx, event := range trx.Events {
						deepmind.TrxBeginEvent(trx.Hash, &event)
						for _, attr := range event.Attributes {
							deepmind.TrxEventAttr(trx.Hash, uint64(idx), attr.Key, attr.Value)
						}
					}
				}
				deepmind.EndBlock(block)
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
