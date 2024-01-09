package core

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/streamingfast/dummy-blockchain/tracer"
	"github.com/streamingfast/dummy-blockchain/types"
)

type Node struct {
	engine Engine
	server Server
	store  *Store
	tracer tracer.Tracer
}

func NewNode(
	storeDir string,
	blockRate int,
	genesisHeight uint64,
	stopHeight uint64,
	serverAddr string,
	tracer tracer.Tracer,
) *Node {
	store := NewStore(storeDir, genesisHeight)

	return &Node{
		engine: NewEngine(genesisHeight, stopHeight, blockRate),
		store:  store,
		server: NewServer(store, serverAddr),
		tracer: tracer,
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
	if tip := node.store.meta.HeadHeight; tip > 0 {
		logrus.WithField("tip", tip).Info("loading last block")
		block, err := node.store.ReadBlock(tip)
		if err != nil {
			logrus.WithError(err).Error("cant read last block")
			return err
		}
		tipBlock = block
	}

	var finalBlock *types.Block
	final := node.store.meta.FinalHeight
	if final == 0 {
		// We are uninitialized, so we need to create a genesis block
		final = node.store.genesisHeight
	}

	logrus.WithField("final", final).Info("loading final block")
	finalBlock, err := node.store.ReadBlock(final)
	if err != nil {
		logrus.WithError(err).Error("cant read final block")
		return err
	}

	if finalBlock == nil {
		return fmt.Errorf("can't find final block %d", final)
	}

	logrus.Info("initializing engine")
	if err := node.engine.Initialize(tipBlock, finalBlock); err != nil {
		logrus.WithError(err).Error("engine initialization failed")
		return err
	}

	if tracer := node.tracer; tracer != nil {
		logrus.Info("initializing tracer")
		if err := node.tracer.Initialize(); err != nil {
			logrus.WithError(err).Error("tracer initialization failed")
			return err
		}
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

			if tracer := node.tracer; tracer != nil {
				tracer.OnBlockStart(block.Header)
				for _, trx := range block.Transactions {
					tracer.OnTrxStart(&trx)

					func() {
						defer tracer.OnTrxEnd(&trx)

						for _, event := range trx.Events {
							tracer.OnTrxEvent(trx.Hash, &event)
						}
					}()
				}

				tracer.OnBlockEnd(block, node.engine.finalBlock.Header)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (node *Node) processBlock(block *types.Block) error {
	logrus.
		WithField("height", block.Header.Height).
		WithField("hash", block.Header.Hash).
		Info("processing block")

	if err := node.store.WriteBlock(block); err != nil {
		return err
	}

	return nil
}
