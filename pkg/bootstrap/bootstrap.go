// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the nonodo run function.
// This is separate from the main package to facilitate testing.
package bootstrap

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/contracts"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/synchronizer"
	synchronizernode "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/synchronizer_node"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/devnet"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/health"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/reader"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/supervisor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

const (
	DefaultHttpPort    = 8080
	DefaultRollupsPort = 5004
	DefaultNamespace   = 10008
)

// Options to nonodo.
type BootstrapOpts struct {
	AutoCount          bool
	HttpAddress        string
	HttpPort           int
	HttpRollupsPort    int
	InputBoxAddress    string
	InputBoxBlock      uint64
	ApplicationAddress string
	// If RpcUrl is set, connect to it instead of anvil.
	RpcUrl      string
	EspressoUrl string
	// If set, start echo dapp.
	EnableEcho bool
	// If set, disables advances.
	DisableAdvance bool
	// If set, disables inspects.
	DisableInspect bool
	// If set, start application.
	ApplicationArgs     []string
	SqliteFile          string
	FromBlock           uint64
	FromBlockL1         *uint64
	DbImplementation    string
	NodeVersion         string
	LoadTestMode        bool
	Namespace           uint64
	TimeoutInspect      time.Duration
	TimeoutWorker       time.Duration
	GraphileUrl         string
	GraphileDisableSync bool
	DbRawUrl            string
	RawEnabled          bool
	EpochBlocks         int
}

// Create the options struct with default values.
func NewBootstrapOpts() BootstrapOpts {
	var (
		defaultTimeout time.Duration = 10 * time.Second
		graphileUrl                  = os.Getenv("GRAPHILE_URL")
	)
	const defaultGraphileUrl = "http://localhost:5001/graphql"

	if graphileUrl == "" {
		graphileUrl = defaultGraphileUrl
	}

	// Check if the URL is valid
	if _, err := url.Parse(graphileUrl); err != nil {
		graphileUrl = defaultGraphileUrl
	}

	return BootstrapOpts{
		HttpAddress:         "127.0.0.1",
		HttpPort:            DefaultHttpPort,
		HttpRollupsPort:     DefaultRollupsPort,
		InputBoxAddress:     devnet.InputBoxAddress,
		InputBoxBlock:       0,
		ApplicationAddress:  devnet.ApplicationAddress,
		RpcUrl:              "",
		EspressoUrl:         "https://query.decaf.testnet.espresso.network",
		EnableEcho:          false,
		DisableAdvance:      false,
		DisableInspect:      false,
		ApplicationArgs:     nil,
		SqliteFile:          "",
		FromBlock:           0,
		FromBlockL1:         nil,
		DbImplementation:    "postgres",
		NodeVersion:         "v1",
		LoadTestMode:        false,
		Namespace:           DefaultNamespace,
		TimeoutInspect:      defaultTimeout,
		TimeoutWorker:       supervisor.DefaultSupervisorTimeout,
		GraphileUrl:         graphileUrl,
		GraphileDisableSync: false,
		AutoCount:           false,
		DbRawUrl:            "postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable",
		RawEnabled:          true,
	}
}

func NewSupervisorGraphQL(opts BootstrapOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker
	w.Timeout = opts.TimeoutWorker
	db := CreateDBInstance(opts)
	container := convenience.NewContainer(*db, opts.AutoCount)
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      opts.TimeoutInspect,
	}))
	health.Register(e)
	reader.Register(e, convenienceService, adapter)
	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpPort),
		Handler: e,
	})

	if opts.RawEnabled {
		dbRawUrl := opts.DbRawUrl
		dbNodeV2 := sqlx.MustConnect("postgres", dbRawUrl)
		rawRepository := synchronizernode.NewRawRepository(opts.DbRawUrl, dbNodeV2)
		synchronizerUpdate := synchronizernode.NewSynchronizerUpdate(
			container.GetRawInputRepository(),
			rawRepository,
			container.GetInputRepository(),
		)
		synchronizerReport := synchronizernode.NewSynchronizerReport(
			container.GetReportRepository(),
			rawRepository,
		)
		synchronizerOutputUpdate := synchronizernode.NewSynchronizerOutputUpdate(
			container.GetVoucherRepository(),
			container.GetNoticeRepository(),
			rawRepository,
			container.GetRawOutputRefRepository(),
		)

		abi, err := contracts.OutputsMetaData.GetAbi()
		if err != nil {
			panic(err)
		}
		abiDecoder := synchronizernode.NewAbiDecoder(abi)

		inputAbi, err := contracts.InputsMetaData.GetAbi()
		if err != nil {
			panic(err)
		}

		inputAbiDecoder := synchronizernode.NewAbiDecoder(inputAbi)

		synchronizerOutputCreate := synchronizernode.NewSynchronizerOutputCreate(
			container.GetVoucherRepository(),
			container.GetNoticeRepository(),
			rawRepository,
			container.GetRawOutputRefRepository(),
			abiDecoder,
		)

		synchronizerOutputExecuted := synchronizernode.NewSynchronizerOutputExecuted(
			container.GetVoucherRepository(),
			container.GetNoticeRepository(),
			rawRepository,
			container.GetRawOutputRefRepository(),
		)

		synchronizerInputCreate := synchronizernode.NewSynchronizerInputCreator(
			container.GetInputRepository(),
			container.GetRawInputRepository(),
			rawRepository,
			inputAbiDecoder,
		)

		rawSequencer := synchronizernode.NewSynchronizerCreateWorker(
			container.GetInputRepository(),
			container.GetRawInputRepository(),
			opts.DbRawUrl,
			rawRepository,
			&synchronizerUpdate,
			container.GetOutputDecoder(),
			synchronizerReport,
			synchronizerOutputUpdate,
			container.GetRawOutputRefRepository(),
			synchronizerOutputCreate,
			synchronizerInputCreate,
			synchronizerOutputExecuted,
		)
		w.Workers = append(w.Workers, rawSequencer)
	}

	cleanSync := synchronizer.NewCleanSynchronizer(container.GetSyncRepository(), nil)
	w.Workers = append(w.Workers, cleanSync)

	slog.Info("Listening", "port", opts.HttpPort)
	return w
}

func NewAbiDecoder(abi *abi.ABI) {
	panic("unimplemented")
}

func CreateDBInstance(opts BootstrapOpts) *sqlx.DB {
	var db *sqlx.DB
	if opts.DbImplementation == "postgres" {
		slog.Info("Using PostGres DB ...")
		postgresHost := os.Getenv("POSTGRES_HOST")
		postgresPort := os.Getenv("POSTGRES_PORT")
		postgresDataBase := os.Getenv("POSTGRES_DB")
		postgresUser := os.Getenv("POSTGRES_USER")
		postgresPassword := os.Getenv("POSTGRES_PASSWORD")
		connectionString := fmt.Sprintf("host=%s port=%s user=%s "+
			"dbname=%s password=%s sslmode=disable",
			postgresHost, postgresPort, postgresUser,
			postgresDataBase, postgresPassword)
		dbUrl, ok := os.LookupEnv("POSTGRES_GRAPHQL_DB_URL")
		if ok {
			connectionString = dbUrl
		} else {
			slog.Warn("The environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, and POSTGRES_PASSWORD are deprecated. Please use POSTGRES_GRAPHQL_DB_URL instead.")
		}
		db = sqlx.MustConnect("postgres", connectionString)
		configureConnectionPool(db)
	} else {
		db = handleSQLite(opts)
	}
	return db
}

// nolint
func configureConnectionPool(db *sqlx.DB) {
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME", 1800) // 30 min
	connMaxIdleTime := getEnvInt("DB_CONN_MAX_IDLE_TIME", 300) // 5 min
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Second)
}

func getEnvInt(envName string, defaultValue int) int {
	value, exists := os.LookupEnv(envName)
	if !exists {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		slog.Error("configuration error", "envName", envName, "value", value)
		panic(err)
	}
	return intValue
}

func handleSQLite(opts BootstrapOpts) *sqlx.DB {
	slog.Info("Using SQLite ...")
	sqliteFile := opts.SqliteFile
	if sqliteFile == "" {
		sqlitePath, err := os.MkdirTemp("", "nonodo-db-*")
		if err != nil {
			panic(err)
		}
		sqliteFile = path.Join(sqlitePath, "nonodo.sqlite3")
		slog.Debug("SQLite3 file created", "path", sqliteFile)
	}

	return sqlx.MustConnect("sqlite3", sqliteFile)
}
