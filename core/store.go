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
	GenesisHash      string `json:"genesis_hash"`
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
	purge        bool

	meta StoreMeta
}

func NewStore(rootDir string, genesisHash string, genesisHeight uint64, genesisTime time.Time, purge bool) *Store {
	return &Store{
		rootDir:      rootDir,
		blocksDir:    filepath.Join(rootDir, "blocks"),
		metaPath:     filepath.Join(rootDir, "meta.json"),
		currentGroup: -1,
		purge:        purge,

		meta: StoreMeta{
			GenesisHash:      genesisHash,
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

	logrus.WithField("dir", store.rootDir).
		WithField("genesis_hash", store.meta.GenesisHash).
		WithField("genesis_height", store.meta.GenesisHeight).
		WithField("genesis_time", time.Unix(0, store.meta.GenesisTimeNanos)).
		WithField("final_height", store.meta.FinalHeight).
		WithField("head_height", store.meta.HeadHeight).
		Info("stored initialized")

	return nil
}

func (store *Store) WriteBlock(block *types.Block) error {
	store.meta.HeadHeight = block.Header.Height
	store.meta.FinalHeight = block.Header.FinalNum

	group := int(store.blockGroup(block.Header.Height))
	if group != store.currentGroup {
		groupDir := fmt.Sprintf("%s/%010d", store.blocksDir, group)
		if err := os.MkdirAll(groupDir, 0700); err != nil {
			return err
		}
		store.currentGroup = group
	}

	file, err := os.Create(store.blockFilename(block.Header.Height))
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(block); err != nil {
		return fmt.Errorf("json encode block: %w", err)
	}

	meta, err := json.MarshalIndent(store.meta, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(store.metaPath, meta, 0655); err != nil {
		return err
	}

	if store.purge {
		if err := store.purgeOldGroups(); err != nil {
			logrus.WithError(err).Warn("failed to purge old block groups")
		}
	}

	return nil
}

func (store *Store) CurrentBlock() (*types.Block, error) {
	return store.ReadBlock(store.meta.HeadHeight)
}

func (store *Store) ReadBlock(height uint64) (*types.Block, error) {
	if height == store.meta.GenesisHeight {
		return types.GenesisBlock(store.meta.GenesisHash, store.meta.GenesisHeight, time.Unix(0, store.meta.GenesisTimeNanos)), nil
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

func (store *Store) purgeOldGroups() error {
	keepGroups := map[uint64]bool{
		store.blockGroup(store.meta.GenesisHeight): true,
		store.blockGroup(store.meta.FinalHeight):   true,
		store.blockGroup(store.meta.HeadHeight):    true,
	}

	entries, err := os.ReadDir(store.blocksDir)
	if err != nil {
		return fmt.Errorf("read blocks dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		var group uint64
		if _, err := fmt.Sscanf(entry.Name(), "%d", &group); err != nil {
			continue
		}

		if keepGroups[group] {
			continue
		}

		groupDir := filepath.Join(store.blocksDir, entry.Name())
		logrus.WithField("dir", groupDir).Debug("purging old block group")
		if err := os.RemoveAll(groupDir); err != nil {
			return fmt.Errorf("remove group dir %s: %w", groupDir, err)
		}
	}

	return nil
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
