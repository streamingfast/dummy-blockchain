package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streamingfast/dummy-blockchain/types"
)

const (
	filesPerDir = 1000
)

type StoreMeta struct {
	GenesisHeight    uint64 `json:"genesis_height"`
	GenesisTimeNanos int64  `json:"genesis_time_nanos"`
	FinalHeight      uint64 `json:"final_height"`
	HeadHeight       uint64 `json:"head_height"`
}

type Store struct {
	rootDir      string
	blocksDir    string
	metaPath     string
	currentGroup int

	meta StoreMeta
}

func NewStore(rootDir string, genesisHeight uint64, genesisTime time.Time) *Store {
	return &Store{
		rootDir:      rootDir,
		blocksDir:    filepath.Join(rootDir, "blocks"),
		metaPath:     filepath.Join(rootDir, "meta.json"),
		currentGroup: -1,

		meta: StoreMeta{
			GenesisHeight:    genesisHeight,
			GenesisTimeNanos: genesisTime.UnixNano(),
		},
	}
}

func (store *Store) Initialize() error {
	logrus.WithField("dir", store.rootDir).Debug("creating store root directory")
	if err := os.MkdirAll(store.rootDir, 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(store.blocksDir, 0700); err != nil {
		return err
	}

	if err := store.readMeta(); err != nil {
		return err
	}

	return nil
}

func (store *Store) WriteBlock(block *types.Block) error {
	store.meta.HeadHeight = block.Header.Height
	store.meta.FinalHeight = block.Header.FinalNum

	raw, err := store.encodeBlock(block)
	if err != nil {
		return err
	}

	meta, err := json.MarshalIndent(store.meta, "", "  ")
	if err != nil {
		return err
	}

	group := int(store.blockGroup(block.Header.Height))
	if group != store.currentGroup {
		groupDir := fmt.Sprintf("%s/%010d", store.blocksDir, group)
		if err := os.MkdirAll(groupDir, 0700); err != nil {
			return err
		}
		store.currentGroup = group
	}

	if err := os.WriteFile(store.blockFilename(block.Header.Height), raw, 0655); err != nil {
		return err
	}

	return os.WriteFile(store.metaPath, meta, 0655)
}

func (store *Store) CurrentBlock() (*types.Block, error) {
	return store.ReadBlock(store.meta.HeadHeight)
}

func (store *Store) ReadBlock(height uint64) (*types.Block, error) {
	if height == store.meta.GenesisHeight {
		return types.GenesisBlock(store.meta.GenesisHeight, time.Unix(0, store.meta.GenesisTimeNanos)), nil
	}

	block := &types.Block{}

	data, err := os.ReadFile(store.blockFilename(height))
	if err != nil {
		return nil, err
	}

	return block, json.Unmarshal(data, block)
}

func (store *Store) blockFilename(height uint64) string {
	return fmt.Sprintf("%s/%010d/%d.json", store.blocksDir, store.blockGroup(height), height)
}

func (store *Store) blockGroup(height uint64) uint64 {
	return height - (height % filesPerDir)
}

func (store *Store) readMeta() error {
	_, err := os.Stat(store.metaPath)
	if err != nil {
		logrus.WithField("path", store.metaPath).WithError(err).Debug("cant open meta file, creating")

		meta, err := json.MarshalIndent(store.meta, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(store.metaPath, meta, 0655); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(store.metaPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &store.meta)
}

func (store *Store) encodeBlock(block *types.Block) ([]byte, error) {
	return json.MarshalIndent(block, "", "  ")
}
