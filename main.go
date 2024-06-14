// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the main function that executes the nonodo command.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/nonodo"
	"github.com/carlmjohnson/versioninfo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var startupMessage = `
Http Rollups for development started at http://localhost:ROLLUPS_PORT
GraphQL running at http://localhost:HTTP_PORT/graphql
Inspect running at http://localhost:HTTP_PORT/inspect/
Press Ctrl+C to stop the node
`
var cmd = &cobra.Command{
	Use:     "nonodo [flags] [-- application [args]...]",
	Short:   "nonodo is a development node for Cartesi Rollups",
	Run:     run,
	Version: versioninfo.Short(),
}

var addressBookCmd = &cobra.Command{
	Use:   "address-book",
	Short: "Show address book",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Read json and print address...")
		devnet.ShowAddresses()
	},
}

// Celestia Network
var celestiaCmd = &cobra.Command{
	Use:   "celestia",
	Short: "Handle blob to Celestia",
	Long:  "Submit a blob and check proofs after one hour to Celestia Network",
}

var (
	debug bool
	color bool
	opts  = nonodo.NewNonodoOpts()
)

func addCelestiaSubcommands(celestiaCmd *cobra.Command) {
	var namespace string
	var height, start, end uint64
	const exampleHeight uint64 = 123456

	// Send
	celestiaSendCmd := &cobra.Command{
		Use:   "send",
		Short: "Send a payload to Celestia Network",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("Send a payload to Celestia Network")
			fmt.Println("Args", args)
		},
	}
	celestiaSendCmd.PersistentFlags().StringVar(&namespace, "namespace", "0xdeadbeef", "Payload to send to Celestia Network")
	celestiaSendCmd.PersistentFlags().Uint64Var(&height, "height", exampleHeight, "Height of the block")
	celestiaSendCmd.PersistentFlags().Uint64Var(&start, "start", 0, "Start of the proof")
	celestiaSendCmd.PersistentFlags().Uint64Var(&end, "end", 1, "End of the proof")
	celestiaCmd.AddCommand(celestiaSendCmd)

	// Check proof
	celestiaCheckProofCmd := &cobra.Command{
		Use:   "check-proof",
		Short: "Check proof of a payload sent to Celestia Network",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("Check proof of a payload sent to Celestia Network")
			fmt.Println("Args", args)
		},
	}
	celestiaCheckProofCmd.PersistentFlags().StringVar(&namespace, "namespace", "0xdeadbeef", "Payload to send to Celestia Network")
	celestiaCheckProofCmd.PersistentFlags().Uint64Var(&height, "height", exampleHeight, "Height of the block")
	celestiaCheckProofCmd.PersistentFlags().Uint64Var(&start, "start", 0, "Start of the proof")
	celestiaCheckProofCmd.PersistentFlags().Uint64Var(&end, "end", 1, "End of the proof")
	celestiaCmd.AddCommand(celestiaCheckProofCmd)

	// Send to relay
	var celestiaRelaySend = &cobra.Command{
		Use:   "relay-send",
		Short: "Send a payload to Celestia Relay",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("Send a payload to Celestia Relay")
			fmt.Println("Args", args)
		}}
	celestiaRelaySend.PersistentFlags().StringVar(&namespace, "namespace", "0xdeadbeef", "Payload to send to Celestia Network")
	celestiaRelaySend.PersistentFlags().Uint64Var(&height, "height", exampleHeight, "Height of the block")
	celestiaRelaySend.PersistentFlags().Uint64Var(&start, "start", 0, "Start of the proof")
	celestiaRelaySend.PersistentFlags().Uint64Var(&end, "end", 1, "End of the proof")
	celestiaCmd.AddCommand(celestiaRelaySend)
}

func init() {
	// anvil-*
	cmd.Flags().StringVar(&opts.AnvilAddress, "anvil-address", opts.AnvilAddress,
		"HTTP address used by Anvil")
	cmd.Flags().IntVar(&opts.AnvilPort, "anvil-port", opts.AnvilPort,
		"HTTP port used by Anvil")
	cmd.Flags().BoolVar(&opts.AnvilVerbose, "anvil-verbose", opts.AnvilVerbose,
		"If set, prints Anvil's output")

	// contracts-*
	cmd.Flags().StringVar(&opts.ApplicationAddress, "contracts-application-address",
		opts.ApplicationAddress, "Application contract address")
	cmd.Flags().StringVar(&opts.InputBoxAddress, "contracts-input-box-address",
		opts.InputBoxAddress, "InputBox contract address")
	cmd.Flags().Uint64Var(&opts.InputBoxBlock, "contracts-input-box-block",
		opts.InputBoxBlock, "InputBox deployment block number")

	// enable-*
	cmd.Flags().BoolVarP(&debug, "enable-debug", "d", false, "If set, enable debug output")
	cmd.Flags().BoolVar(&color, "enable-color", true, "If set, enables logs color")
	cmd.Flags().BoolVar(&opts.EnableEcho, "enable-echo", opts.EnableEcho,
		"If set, nonodo starts a built-in echo application")

	cmd.Flags().StringVar(&opts.Sequencer, "sequencer", opts.Sequencer,
		"Set the sequencer (inputbox[default] or espresso)")
	cmd.Flags().Uint64Var(&opts.Namespace, "namespace", opts.Namespace,
		"Set the namespace for espresso)")

	// disable-*
	cmd.Flags().BoolVar(&opts.DisableDevnet, "disable-devnet", opts.DisableDevnet,
		"If set, nonodo won't start a local devnet")
	cmd.Flags().BoolVar(&opts.DisableAdvance, "disable-advance", opts.DisableAdvance,
		"If set, nonodo won't start the inputter to get inputs from the local chain")

	// http-*
	cmd.Flags().StringVar(&opts.HttpAddress, "http-address", opts.HttpAddress,
		"HTTP address used by nonodo to serve its APIs")
	cmd.Flags().IntVar(&opts.HttpPort, "http-port", opts.HttpPort,
		"HTTP port used by nonodo to serve its external APIs")
	cmd.Flags().IntVar(&opts.HttpRollupsPort, "http-rollups-port", opts.HttpRollupsPort,
		"HTTP port used by nonodo to serve its internal APIs")

	// rpc-url
	cmd.Flags().StringVar(&opts.RpcUrl, "rpc-url", opts.RpcUrl,
		"If set, nonodo connects to this url instead of setting up Anvil")

	// convenience experimental implementation
	cmd.Flags().BoolVar(&opts.HLGraphQL, "high-level-graphql", opts.HLGraphQL,
		"If set, enables the convenience layer experiment")

	// database file
	cmd.Flags().StringVar(&opts.SqliteFile, "sqlite-file", opts.SqliteFile,
		"The sqlite file to load the state")

	cmd.Flags().Uint64Var(&opts.FromBlock, "from-block", opts.FromBlock,
		"The beginning of the queried range for events")

	cmd.Flags().StringVar(&opts.DbImplementation, "db-implementation", opts.DbImplementation,
		"DB to use. PostgreSQL or SQLite")

	cmd.Flags().StringVar(&opts.NodeVersion, "node-version", opts.NodeVersion,
		"Node version to emulate")

	cmd.Flags().BoolVar(&opts.LoadTestMode, "load-test-mode", opts.LoadTestMode,
		"If set, enables load test mode")
}

func run(cmd *cobra.Command, args []string) {
	startTime := time.Now()

	// setup log
	logOpts := new(tint.Options)
	if debug {
		logOpts.Level = slog.LevelDebug
	}
	logOpts.AddSource = debug
	logOpts.NoColor = !color || !isatty.IsTerminal(os.Stdout.Fd())
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// check args
	checkEthAddress(cmd, "address-input-box")
	checkEthAddress(cmd, "address-application")
	if opts.AnvilPort == 0 {
		exitf("--anvil-port cannot be 0")
	}
	if cmd.Flags().Changed("rpc-url") && !cmd.Flags().Changed("contracts-input-box-block") {
		exitf("must set --contracts-input-box-block when setting --rpc-url")
	}
	if opts.EnableEcho && len(args) > 0 {
		exitf("can't use built-in echo with custom application")
	}
	opts.ApplicationArgs = args

	// handle signals with notify context
	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// start nonodo
	ready := make(chan struct{}, 1)
	go func() {
		select {
		case <-ready:
			msg := strings.Replace(
				startupMessage,
				"HTTP_PORT",
				fmt.Sprint(opts.HttpPort), -1)
			msg = strings.Replace(
				msg,
				"ROLLUPS_PORT",
				fmt.Sprint(opts.HttpRollupsPort), -1)
			fmt.Println(msg)
			slog.Info("nonodo: ready", "after", time.Since(startTime))
		case <-ctx.Done():
		}
	}()
	if opts.HLGraphQL {
		err := nonodo.NewSupervisorPoC(opts).Start(ctx, ready)
		cobra.CheckErr(err)
	} else {
		err := nonodo.NewSupervisor(opts).Start(ctx, ready)
		cobra.CheckErr(err)
	}

}

func main() {
	addCelestiaSubcommands(celestiaCmd)
	cmd.AddCommand(addressBookCmd, celestiaCmd)
	cobra.CheckErr(cmd.Execute())
}

func exitf(format string, args ...any) {
	err := fmt.Sprintf(format, args...)
	slog.Error("configuration error", "error", err)
	os.Exit(1)
}

func checkEthAddress(cmd *cobra.Command, varName string) {
	if cmd.Flags().Changed(varName) {
		value, err := cmd.Flags().GetString(varName)
		cobra.CheckErr(err)
		bytes, err := hexutil.Decode(value)
		if err != nil {
			exitf("invalid address for --%v: %v", varName, err)
		}
		if len(bytes) != common.AddressLength {
			exitf("invalid address for --%v: wrong length", varName)
		}
	}
}
