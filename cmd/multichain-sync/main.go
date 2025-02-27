package main

import (
	"context"
	"os"

	"github.com/ethereum/go-ethereum/log"

	"github.com/JokingLove/multichain-sync-account/common/opio"
)

var (
	GitCommit = ""
	GitData   = ""
)

func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stdout, log.LevelInfo, true)))
	app := NewCli(GitCommit, GitData)
	ctx := opio.WithInterruptBlocker(context.Background())

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error("Application failed", "error", err)
		os.Exit(1)
	}

}
