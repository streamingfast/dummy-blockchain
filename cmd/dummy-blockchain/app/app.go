package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
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
	GenesisBlockBurst uint64
	LogLevel          string
	StoreDir          string
	BlockRate         int
	BlockSizeInBytes  int
	ServerAddr        string
	Tracer            string
	StopHeight        uint64

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
	flags.IntVar(&cliOpts.BlockSizeInBytes, "block-size", 64000, "Approximate block size (in bytes) to produce")
	flags.Uint64Var(&cliOpts.StopHeight, "stop-height", 0, "Stop block production at this height")
	flags.StringVar(&cliOpts.ServerAddr, "server-addr", "0.0.0.0:8080", "Server address")
	flags.StringVar(&cliOpts.Tracer, "tracer", "", "The tracer to use, either <empty>, none or firehose")

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

			node := core.NewNode(
				cliOpts.StoreDir,
				cliOpts.BlockRate,
				cliOpts.BlockSizeInBytes,
				GenesisHash,
				GenesisHeight,
				genesisTime,
				cliOpts.GenesisBlockBurst,
				cliOpts.StopHeight,
				cliOpts.ServerAddr,
				blockTracer,
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
