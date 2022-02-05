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
	GenesisHeight uint64 `long:"genesis-height" description:"Blockhain genesis height" default:"1"`
	LogLevel      string `long:"log-level" description:"Logging level" default:"info"`
	StoreDir      string `long:"store-dir" description:"Directory for storing blocks data" default:"./data"`
	BlockRate     int    `long:"block-rate" description:"Block production rate (per second)" default:"1"`
}{}

func main() {
	root := cobra.Command{
		Use:   "dummy-chain",
		Short: "CLI for the Dummy Chain",
	}

	root.PersistentFlags().Uint64Var(&cliOpts.GenesisHeight, "genesis-height", 1, "Blockchain genesis height")
	root.PersistentFlags().StringVar(&cliOpts.LogLevel, "log-level", "info", "Logging level")
	root.PersistentFlags().StringVar(&cliOpts.StoreDir, "store-dir", "./data", "Directory for storing blockchain state")
	root.PersistentFlags().IntVar(&cliOpts.BlockRate, "block-rate", 1, "Block production rate (per second)")

	if err := root.ParseFlags(os.Args); err != nil {
		logrus.Fatal(err)
	}

	level, err := logrus.ParseLevel(cliOpts.LogLevel)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	root.AddCommand(
		makeInitCommand(),
		makeResetCommand(),
		makeStartComand(),
	)

	root.Execute()
}

func makeInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize local blockchain state",
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.WithField("dir", cliOpts.StoreDir).Info("initializing chain store")

			store := core.NewStore(cliOpts.StoreDir)
			return store.Initialize()
		},
	}
}

func makeResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset local blockchain state",
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
		Use:   "start",
		Short: "Start blockchian service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliOpts.BlockRate < 1 {
				return errors.New("block rate option must be greater than 1")
			}

			// TODO: expose this as a flag too
			if os.Getenv("DM_ENABLED") == "1" {
				initDeepMind()
				defer deepmind.Shutdown()
			}

			node := core.NewNode(
				cliOpts.StoreDir,
				cliOpts.BlockRate,
				cliOpts.GenesisHeight,
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
