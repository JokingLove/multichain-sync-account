package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	multichain_transaction_syncs "github.com/JokingLove/multichain-sync-account"
	"github.com/JokingLove/multichain-sync-account/common/cliapp"
	"github.com/JokingLove/multichain-sync-account/common/opio"
	"github.com/JokingLove/multichain-sync-account/config"
	"github.com/JokingLove/multichain-sync-account/database"
	flags2 "github.com/JokingLove/multichain-sync-account/flags"
	"github.com/JokingLove/multichain-sync-account/notifier"
	"github.com/JokingLove/multichain-sync-account/rpcclient"
	"github.com/JokingLove/multichain-sync-account/rpcclient/chain-account/account"
	"github.com/JokingLove/multichain-sync-account/services"
)

func NewCli(GitCommit string, GitData string) *cli.App {
	flags := flags2.Flags
	return &cli.App{
		Version:              "1.0.1", // params.VersionWithCommit(GitCommit, GitData),
		Description:          "An exchange wallet scanner services with rpc and rest api server",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "rpc",
				Flags:       flags,
				Description: "Run rpc service",
				Action:      cliapp.LifecycleCmd(runRpc),
			},
			{
				Name:        "sync",
				Flags:       flags,
				Description: "Run rpc scanner wallet chain node",
				Action:      cliapp.LifecycleCmd(runMultichainSync),
			},
			{
				Name:        "migrate",
				Flags:       flags,
				Description: "Run database migration",
				Action:      runMigrations,
			},
			{
				Name:        "notify",
				Flags:       flags,
				Description: "Run notify service",
				Action:      cliapp.LifecycleCmd(runNotify),
			},
			{
				Name:        "version",
				Description: "Show project version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}

func runNotify(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("running notify task ...... ")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("load config failed", "err", err)
		return nil, err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return nil, err
	}
	return notifier.NewNotifier(db, shutdown)
}

func runMigrations(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("running migrations.....")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			log.Error("failed to close database", "err", err)
		}
	}(db)

	return db.ExecuteSQLMigration(cfg.Migrations)
}

func runMultichainSync(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("exec wallet sync")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "error", err)
		return nil, err
	}
	return multichain_transaction_syncs.NewMultiChainSync(ctx.Context, &cfg, shutdown)
}

func runRpc(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	fmt.Println("running rpc server...")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}

	grpcServerCfg := &services.BusinessMiddleConfig{
		GrpcHostName: cfg.RpcServer.Host,
		GrpcPort:     cfg.RpcServer.Port,
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return nil, err
	}

	log.Info("Chain account rpc ", "rpc uri", cfg.ChainAccountRpc)
	conn, err := grpc.NewClient(cfg.ChainAccountRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Connect to da retriever failed", "err", err)
		return nil, err
	}

	client := account.NewWalletAccountServiceClient(conn)
	accountClient, err := rpcclient.NewWalletChainAccountClient(context.Background(), client, "Ethereum")
	if err != nil {
		log.Error("new wallet account client failed", "err", err)
		return nil, err
	}

	return services.NewBusinessMiddleWireServices(db, grpcServerCfg, accountClient)
}
