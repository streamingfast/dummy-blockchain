package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/streamingfast/dummy-blockchain/types"
)

type Store struct {
	rootDir   string
	blocksDir string
	metaPath  string

	meta struct {
		StartHeight uint64 `json:"start_height"`
		TipHeight   uint64 `json:"tip_height"`
	}
}

func NewStore(rootDir string) *Store {
	return &Store{
		rootDir:   rootDir,
		blocksDir: filepath.Join(rootDir, "blocks"),
		metaPath:  filepath.Join(rootDir, "meta.json"),
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
	store.meta.TipHeight = block.Height
	if store.meta.StartHeight == 0 {
		store.meta.StartHeight = block.Height
	}

	raw, err := store.encodeBlock(block)
	if err != nil {
		return err
	}

	meta, err := json.MarshalIndent(store.meta, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(store.blockFilename(block.Height), raw, 0655); err != nil {
		return err
	}

	return ioutil.WriteFile(store.metaPath, meta, 0655)
}

func (store *Store) CurrentBlock() (*types.Block, error) {
	return store.ReadBlock(store.meta.TipHeight)
}

func (store *Store) ReadBlock(height uint64) (*types.Block, error) {
	block := &types.Block{}

	data, err := ioutil.ReadFile(store.blockFilename(height))
	if err != nil {
		return nil, err
	}

	return block, json.Unmarshal(data, block)
}

func (store *Store) blockFilename(height uint64) string {
	return fmt.Sprintf("%s/%d.json", store.blocksDir, height)
}

func (store *Store) readMeta() error {
	_, err := os.Stat(store.metaPath)
	if err != nil {
		logrus.WithField("path", store.metaPath).WithError(err).Debug("cant open meta file, creating")

		meta, err := json.MarshalIndent(store.meta, "", "  ")
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(store.metaPath, meta, 0655); err != nil {
			return err
		}
	}

	data, err := ioutil.ReadFile(store.metaPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &store.meta)
}

func (store *Store) encodeBlock(block *types.Block) ([]byte, error) {
	return json.MarshalIndent(block, "", "  ")
}
