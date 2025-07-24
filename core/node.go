package core

import (
	"context"
	"fmt"
	"time"

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
	genesisHash string,
	genesisHeight uint64,
	genesisTime time.Time,
	genesisBlockBurst uint64,
	stopHeight uint64,
	serverAddr string,
	tracer tracer.Tracer,
) *Node {
	store := NewStore(storeDir, genesisHash, genesisHeight, genesisTime)

	return &Node{
		engine: NewEngine(genesisHash, genesisHeight, genesisTime, genesisBlockBurst, stopHeight, blockRate),
		store:  store,
		server: NewServer(store, serverAddr),
		tracer: tracer,
	}
}

func (node *Node) Initialize() error {
	logrus.
		WithField("genesis_height", node.store.meta.GenesisHeight).
		Info("initializing node")

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
		final = node.store.meta.GenesisHeight
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
		WithField("block", blockRef{block.Header.Hash, block.Header.Height}).
		WithField("parent_block", blockRef{valueOr(block.Header.PrevHash, ""), valueOr(block.Header.PrevNum, 0)}).
		WithField("final_block", blockRef{block.Header.FinalHash, block.Header.FinalNum}).
		Info("processing block")

	if err := node.store.WriteBlock(block); err != nil {
		return err
	}

	return nil
}

func shortHash(in string) string {
	return in[:6] + "..." + in[len(in)-6:]
}

func shortHashPtr(in *string) string {
	if in == nil {
		return "<nil>"
	}

	return shortHash(*in)
}

func valueOr[T any](t *T, def T) T {
	if t == nil {
		return def
	}

	return *t
}

type blockRef struct {
	hash   string
	number uint64
}

func (ref blockRef) String() string {
	if ref.hash == "" {
		return "<nil>"
	}

	return fmt.Sprintf("#%d (%s)", ref.number, shortHash(ref.hash))
}
