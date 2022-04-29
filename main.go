package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/streamingfast/dummy-blockchain/core"
	"github.com/streamingfast/dummy-blockchain/deepmind"
)

var cliOpts = struct {
	GenesisHeight   uint64
	LogLevel        string
	StoreDir        string
	BlockRate       int
	ServerAddr      string
	Instrumentation bool
	StopHeight      uint64
}{}

func main() {
	root := &cobra.Command{
		Use:     "dummy-blockchain",
		Short:   "CLI for the Dummy Chain",
		Version: VersionString(),
	}

	root.SetOutput(os.Stderr)

	if err := initFlags(root); err != nil {
		logrus.Fatal(err)
	}

	if err := initLogger(); err != nil {
		logrus.Fatal(err)
	}

	root.AddCommand(
		makeInitCommand(),
		makeResetCommand(),
		makeStartComand(),
	)

	root.Execute()
}

func initFlags(root *cobra.Command) error {
	flags := root.PersistentFlags()

	flags.Uint64Var(&cliOpts.GenesisHeight, "genesis-height", 1, "Blockchain genesis height")
	flags.StringVar(&cliOpts.LogLevel, "log-level", "info", "Logging level")
	flags.StringVar(&cliOpts.StoreDir, "store-dir", "./data", "Directory for storing blockchain state")
	flags.IntVar(&cliOpts.BlockRate, "block-rate", 60, "Block production rate (per minute)")
	flags.Uint64Var(&cliOpts.StopHeight, "stop-height", 0, "Stop block production at this height")
	flags.StringVar(&cliOpts.ServerAddr, "server-addr", "0.0.0.0:8080", "Server address")
	flags.BoolVar(&cliOpts.Instrumentation, "dm-enabled", false, "Enable instrumentation")

	return nil
}

func initLogger() error {
	level, err := logrus.ParseLevel(cliOpts.LogLevel)
	if err != nil {
		return err
	}

	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return nil
}

func makeInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "Initialize local blockchain state",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.WithField("dir", cliOpts.StoreDir).Info("initializing chain store")

			store := core.NewStore(cliOpts.StoreDir)
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
			if cliOpts.BlockRate < 1 {
				return errors.New("block rate option must be greater than 1")
			}

			if cliOpts.Instrumentation || os.Getenv("DM_ENABLED") == "1" {
				initDeepMind()
				defer deepmind.Shutdown()
			}

			node := core.NewNode(
				cliOpts.StoreDir,
				cliOpts.BlockRate,
				cliOpts.GenesisHeight,
				cliOpts.StopHeight,
				cliOpts.ServerAddr,
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

func initDeepMind() {
	// A global flag to enable instrumentation
	dmOutput := os.Getenv("DM_OUTPUT")

	switch dmOutput {
	case "", "stdout", "STDOUT":
		deepmind.Enable(os.Stdout)
	case "stderr", "STDERR":
		deepmind.Enable(os.Stderr)
	default:
		dmFile, err := os.OpenFile(dmOutput, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0666)
		if err != nil {
			logrus.WithError(err).Fatal("cant open DM output file")
		}
		deepmind.Enable(dmFile)
	}
}

func waitForSignal() os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGINT)
	return <-sig
}
