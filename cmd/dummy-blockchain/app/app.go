package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/streamingfast/dummy-blockchain/core"
	"github.com/streamingfast/dummy-blockchain/tracer"
)

const (
	GenesisHash   string = "0x0000000000000000000000000000000000000000000000000000000000000000"
	GenesisHeight uint64 = 0
)

type Flags struct {
	GenesisBlockBurst    uint64
	LogLevel             string
	StoreDir             string
	BlockRate            int
	BlockSize            string
	ServerAddr           string
	WithCommitmentSignal bool
	WithSkippedBlocks    bool
	WithReorgs           bool
	WithFlashBlocks      bool
	Tracer               string
	StopHeight           uint64

	Deprecated struct {
		GenesisHeight  uint64
		GenesisTimeRaw string
	}
}

var cliOpts Flags

func Main(version string) {
	root := &cobra.Command{
		Use:     "dummy-blockchain",
		Short:   "CLI for the Dummy Chain",
		Version: VersionString(version),
	}

	root.SetOutput(os.Stderr)

	if err := initFlags(root); err != nil {
		logrus.Fatal(err)
	}

	root.AddCommand(
		makeInitCommand(),
		makeResetCommand(),
		makeStartComand(),
	)

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return initLogger()
	}

	root.Execute()
}

func initFlags(root *cobra.Command) error {
	flags := root.PersistentFlags()

	flags.Uint64Var(&cliOpts.Deprecated.GenesisHeight, "genesis-height", 0, "Deprecated: The height of the genesis block, ignored, hard-coded to 0")
	flags.StringVar(&cliOpts.Deprecated.GenesisTimeRaw, "genesis-time", "", "Deprecated: The time of the genesis block, ignored, hard-coded to current time")
	flags.Uint64Var(&cliOpts.GenesisBlockBurst, "genesis-block-burst", 0, "The amount of block to produce when initially starting from genesis block")
	flags.StringVar(&cliOpts.LogLevel, "log-level", "info", "Logging level")
	flags.StringVar(&cliOpts.StoreDir, "store-dir", "./data", "Directory for storing blockchain state")
	flags.IntVar(&cliOpts.BlockRate, "block-rate", 60, "Block production rate (per minute)")
	flags.StringVar(&cliOpts.BlockSize, "block-size", "64 KiB", "Approximate block size (in bytes) to produce, accepts integere (with _) or human-readable sizes (e.g. 64KiB, 2 MiB)")
	flags.Uint64Var(&cliOpts.StopHeight, "stop-height", 0, "Stop block production at this height")
	flags.StringVar(&cliOpts.ServerAddr, "server-addr", "0.0.0.0:8080", "Server address")
	flags.StringVar(&cliOpts.Tracer, "tracer", "", "The tracer to use, either <empty>, none or firehose")
	flags.BoolVar(&cliOpts.WithCommitmentSignal, "with-signal", false, "Whether we produce BlockCommitmentLevel signals on top of blocks")
	flags.BoolVar(&cliOpts.WithFlashBlocks, "with-flash-blocks", false, "Whether we produce 4 flash blocks per block, skipping number 2 every 11 slots")
	flags.BoolVar(&cliOpts.WithSkippedBlocks, "with-skipped-blocks", true, "Whether we skip a block number every 13 slots")
	flags.BoolVar(&cliOpts.WithReorgs, "with-reorgs", true, "Whether we produce reorgs every 17 slots")

	return nil
}

func initLogger() error {
	level, err := logrus.ParseLevel(cliOpts.LogLevel)
	if err != nil {
		return err
	}

	logrus.SetLevel(level)
	logrus.SetOutput(os.Stderr)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return nil
}

func makeInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "Initialize local blockchain state",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			warnDeprecatedFlags()

			genesisTime := time.Now()

			logrus.
				WithField("dir", cliOpts.StoreDir).
				Info("initializing chain store")

			store := core.NewStore(cliOpts.StoreDir, GenesisHash, GenesisHeight, genesisTime)
			return store.Initialize()
		},
	}
}

func makeResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "reset",
		Short:        "Reset local blockchain state",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.WithField("dir", cliOpts.StoreDir).Info("removing chain store")

			err := os.RemoveAll(cliOpts.StoreDir)
			if err != nil {
				logrus.WithError(err).Error("cant remove the chain store directory")
			}

			return err
		},
	}
}

func makeStartComand() *cobra.Command {
	return &cobra.Command{
		Use:          "start",
		Short:        "Start blockchain service",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			warnDeprecatedFlags()

			if cliOpts.BlockRate < 1 {
				return errors.New("block rate option must be greater than 1")
			}

			genesisTime := time.Now()

			logrus.
				WithField("dir", cliOpts.StoreDir).
				Info("starting chain service")

			var blockTracer tracer.Tracer
			if cliOpts.Tracer == "firehose" {
				blockTracer = &tracer.FirehoseTracer{}
			}

			blockSizeInBytes := 64 * 1024 // Default to 64 KiB
			if cliOpts.BlockSize != "" {
				parsedSize, err := parseByteSize(cliOpts.BlockSize)
				if err != nil {
					return err
				}

				blockSizeInBytes = int(parsedSize)
			}

			node := core.NewNode(
				cliOpts.StoreDir,
				cliOpts.BlockRate,
				blockSizeInBytes,
				GenesisHash,
				GenesisHeight,
				genesisTime,
				cliOpts.GenesisBlockBurst,
				cliOpts.StopHeight,
				cliOpts.ServerAddr,
				blockTracer,
				cliOpts.WithCommitmentSignal,
				cliOpts.WithSkippedBlocks,
				cliOpts.WithReorgs,
				cliOpts.WithFlashBlocks,
			)

			if err := node.Initialize(); err != nil {
				logrus.WithError(err).Fatal("node failed to initialize")
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				sig := waitForSignal()
				logrus.WithField("signal", sig).Info("shutting down")
				cancel()
			}()

			if err := node.Start(ctx); err != nil {
				logrus.WithError(err).Fatal("node terminated with error")
			} else {
				logrus.Info("node terminated")
			}

			return nil
		},
	}
}

func warnDeprecatedFlags() {
	if cliOpts.Deprecated.GenesisHeight != 0 {
		logrus.Warn("the --genesis-height flag is deprecated and ignored, the genesis height is hard-coded to 0")
	}

	if cliOpts.Deprecated.GenesisTimeRaw != "" {
		logrus.Warn("the --genesis-time flag is deprecated and ignored, the genesis time is hard-coded to current time")
	}
}

func waitForSignal() os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGINT)
	return <-sig
}

var integerRegex = regexp.MustCompile(`^[0-9_]+$`)
var humanReadableRegex = regexp.MustCompile(`(?i)^([0-9_]+)\s*(kib|mib|gib|tib|kb|mb|gb|tb)$`)
var unitToMultiplier = map[string]uint64{
	"kb":  1000,
	"mb":  1000 * 1000,
	"gb":  1000 * 1000 * 1000,
	"tb":  1000 * 1000 * 1000 * 1000,
	"kib": 1024,
	"mib": 1024 * 1024,
	"gib": 1024 * 1024 * 1024,
	"tib": 1024 * 1024 * 1024 * 1024,
}

func parseByteSize(sizeStr string) (uint64, error) {
	if integerRegex.MatchString(sizeStr) {
		size, err := strconv.ParseUint(sizeStr, 0, 64)
		if err != nil {
			return 0, err
		}
		return size, nil
	}

	matches := humanReadableRegex.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid byte size format %q", sizeStr)
	}

	numberPart := matches[1]
	unitPart := strings.ToLower(matches[2])

	number, err := strconv.ParseUint(numberPart, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in byte size: %w", err)
	}

	multiplier, found := unitToMultiplier[unitPart]
	if !found {
		return 0, fmt.Errorf("unknown unit in byte size: %q", unitPart)
	}

	return number * multiplier, nil
}
