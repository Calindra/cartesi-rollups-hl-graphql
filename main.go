// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the main function that executes the hlgraphql command.
package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/bootstrap"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/devnet"
	"github.com/carlmjohnson/versioninfo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	MAX_FILE_SIZE uint64 = 1_440_000 // 1,44 MB
	APP_ADDRESS          = common.HexToAddress(devnet.ApplicationAddress)
)

var inspectMessageText = `
Inspect running at http://localhost:HTTP_PORT/inspect/`

var startupMessage = `
Http Rollups for development started at http://localhost:ROLLUPS_PORT
GraphQL running at http://localhost:HTTP_PORT/graphql
Press Ctrl+C to stop the node
`

var tempFromBlockL1 uint64

var cmd = &cobra.Command{
	Use:     "hlgraphql [flags] [-- application [args]...]",
	Short:   "hlgraphql is a development node for Cartesi Rollups",
	Run:     run,
	Version: versioninfo.Short(),
}

var CompletionCmd = &cobra.Command{
	Use:                   "completion",
	Short:                 "Generate shell completion scripts",
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cobra.CheckErr(cmd.Root().GenBashCompletion(os.Stdout))
		case "zsh":
			cobra.CheckErr(cmd.Root().GenZshCompletion(os.Stdout))
		case "fish":
			cobra.CheckErr(cmd.Root().GenFishCompletion(os.Stdout, true))
		case "powershell":
			cobra.CheckErr(cmd.Root().GenPowerShellCompletion(os.Stdout))
		}
	},
}

// Celestia Network
type CelestiaOpts struct {
	Payload     string
	PayloadPath string
	PayloadUrl  string
	Namespace   string
	Height      uint64
	Start       uint64
	End         uint64
	RpcUrl      string
}

// Espresso
type EspressoOpts struct {
	Payload   string
	Namespace int
}

var celestiaCmd = &cobra.Command{
	Use:   "celestia",
	Short: "Handle blob to Celestia",
	Long:  "Submit a blob and check proofs after one hour to Celestia Network",
}

var espressoCmd = &cobra.Command{
	Use:   "espresso",
	Short: "Handles Espresso transactions",
	Long:  "Submit and get a transaction from Espresso using Cappuccino APIs",
}

type AvailOpts struct {
	Payload     string
	ChainId     int
	AppId       int
	Address     string
	MaxGasPrice uint64
}

var availCmd = &cobra.Command{
	Use:   "avail",
	Short: "Handles Avail transactions",
	Long:  "Submit a transaction to Avail",
}

var (
	debug bool
	color bool
	opts  = bootstrap.NewBootstrapOpts()
)

func ArrBytesAttr(key string, v []byte) slog.Attr {
	var str string
	for _, b := range v {
		s := fmt.Sprintf("%02x", b)
		str += s
	}
	return slog.String(key, str)
}

func CheckIfValidSize(size uint64) error {
	if size > MAX_FILE_SIZE {
		return fmt.Errorf("file size is too big %d bytes", size)
	}

	return nil
}

func init() {
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
		"If set, hlgraphql starts a built-in echo application")

	cmd.Flags().StringVar(&opts.EspressoUrl, "espresso-url", opts.EspressoUrl,
		"Set the Espresso base url")

	cmd.Flags().Uint64Var(&opts.Namespace, "namespace", opts.Namespace,
		"Set the namespace for espresso")
	cmd.Flags().DurationVar(&opts.TimeoutWorker, "timeout-worker", opts.TimeoutWorker, "Timeout for workers. Example: hlgraphql --timeout-worker 30s")
	cmd.Flags().DurationVar(&opts.TimeoutInspect, "sm-deadline-inspect-state", opts.TimeoutInspect, "Timeout for inspect requests. Example: hlgraphql --sm-deadline-inspect-state 30s")
	cmd.Flags().DurationVar(&opts.TimeoutAdvance, "sm-deadline-advance-state", opts.TimeoutAdvance, "Timeout for advance requests. Example: hlgraphql --sm-deadline-advance-state 30s")

	// disable-*
	cmd.Flags().BoolVar(&opts.DisableAdvance, "disable-advance", opts.DisableAdvance,
		"If set, hlgraphql won't start the inputter to get inputs from the local chain")
	cmd.Flags().BoolVar(&opts.DisableInspect, "disable-inspect", opts.DisableInspect,
		"If set, hlgraphql won't accept inspect inputs")

	// http-*
	cmd.Flags().StringVar(&opts.HttpAddress, "http-address", opts.HttpAddress,
		"HTTP address used by hlgraphql to serve its APIs")
	cmd.Flags().IntVar(&opts.HttpPort, "http-port", opts.HttpPort,
		"HTTP port used by hlgraphql to serve its external APIs")
	cmd.Flags().IntVar(&opts.HttpRollupsPort, "http-rollups-port", opts.HttpRollupsPort,
		"HTTP port used by hlgraphql to serve its internal APIs")

	// rpc-url
	cmd.Flags().StringVar(&opts.RpcUrl, "rpc-url", opts.RpcUrl,
		"If set, hlgraphql connects to this url instead of setting up Anvil")

	// convenience experimental implementation
	cmd.Flags().BoolVar(&opts.GraphQL, "graphql", opts.GraphQL,
		"If set, enables the graphql layer")

	// database file
	cmd.Flags().StringVar(&opts.SqliteFile, "sqlite-file", opts.SqliteFile,
		"The sqlite file to load the state")

	cmd.Flags().Uint64Var(&opts.FromBlock, "from-block", opts.FromBlock,
		"The beginning of the queried range for events")

	cmd.Flags().Uint64VarP(&tempFromBlockL1, "from-l1-block", "", 0, "The beginning of the queried range for events")

	cmd.Flags().StringVar(&opts.DbImplementation, "db-implementation", opts.DbImplementation,
		"DB to use. PostgreSQL or SQLite")

	cmd.Flags().StringVar(&opts.NodeVersion, "node-version", opts.NodeVersion,
		"Node version to emulate")

	cmd.Flags().BoolVar(&opts.LoadTestMode, "load-test-mode", opts.LoadTestMode,
		"If set, enables load test mode")

	cmd.Flags().BoolVar(&opts.GraphileDisableSync, "graphile-disable-sync", opts.GraphileDisableSync,
		"If set, disable graphile synchronization")

	cmd.Flags().StringVar(&opts.GraphileUrl, "graphile-url", opts.GraphileUrl, "URL used to connect to Graphile")

	cmd.Flags().StringVar(&opts.DbRawUrl, "db-raw-url", opts.DbRawUrl, "The raw database url")
	cmd.Flags().BoolVar(&opts.RawEnabled, "raw-enabled", opts.RawEnabled, "If set, enables raw database")

	cmd.Flags().IntVar(&opts.EpochBlocks, "epoch-blocks", opts.EpochBlocks,
		"Number of blocks in each epoch")

}

func deprecatedWarning(flag string, replacement string) {
	if strings.Contains(strings.Join(os.Args, " "), "--"+flag) {
		slog.Warn(fmt.Sprintf("The '%s' flag is deprecated. %s", flag, replacement))
	}
}

func run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
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
	if !cmd.Flags().Changed("sequencer") && cmd.Flags().Changed("rpc-url") && !cmd.Flags().Changed("contracts-input-box-block") {
		exitf("must set --contracts-input-box-block when setting --rpc-url")
	}
	if opts.EnableEcho && len(args) > 0 {
		exitf("can't use built-in echo with custom application")
	}
	if cmd.Flags().Changed("from-l1-block") {
		opts.FromBlockL1 = &tempFromBlockL1
	}
	deprecatedWarning("high-level-graphql", "")
	deprecatedWarning("graphile-disable-sync", "")
	deprecatedWarning("disable-devnet", "")
	deprecatedWarning("db-raw-url", "Please use POSTGRES_NODE_DB_URL instead.")
	deprecatedWarning("sequencer", "")
	deprecatedWarning("salsa", "")
	deprecatedWarning("salsa-url", "")
	deprecatedWarning("avail-enabled", "")
	deprecatedWarning("avail-from-block", "")
	deprecatedWarning("paio-server-url", "")

	opts.ApplicationArgs = args

	// handle signals with notify context
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	startMessage := startupMessage

	var inspectMessage string
	if !opts.DisableInspect {
		inspectMessage = inspectMessageText
	}

	// start hlgraphql
	ready := make(chan struct{}, 1)
	go func() {
		select {
		case <-ready:
			msg := strings.Replace(
				startMessage,
				"\nINSPECT_MESSAGE",
				inspectMessage, -1)
			msg = strings.Replace(
				msg,
				"HTTP_PORT",
				fmt.Sprint(opts.HttpPort), -1)
			msg = strings.Replace(
				msg,
				"ROLLUPS_PORT",
				fmt.Sprint(opts.HttpRollupsPort), -1)
			fmt.Println(msg)
			slog.Info("hlgraphql: ready", "after", time.Since(startTime))
		case <-ctx.Done():
		}
	}()
	LoadEnv()
	var err error = bootstrap.NewSupervisorGraphQL(opts).Start(ctx, ready)
	cobra.CheckErr(err)
}

//go:embed .env
var envBuilded string

// LoadEnv from embedded .env file
func LoadEnv() {
	currentEnv := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		currentEnv[key] = true
	}

	parse, err := godotenv.Unmarshal(envBuilded)
	cobra.CheckErr(err)

	for k, v := range parse {
		if !currentEnv[k] {
			slog.Debug("env: setting env", "key", k, "value", v)
			err := os.Setenv(k, v)
			cobra.CheckErr(err)
		} else {
			slog.Debug("env: skipping env", "key", k)
		}
	}

	slog.Debug("env: loaded")
}

func main() {
	cmd.AddCommand(celestiaCmd, CompletionCmd, espressoCmd, availCmd)
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
