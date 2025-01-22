package rpcclient

import (
	"context"
	"github.com/JokingLove/multichain-sync-account/config"
	"github.com/JokingLove/multichain-sync-account/database"
)

const (
	notifyUrl        = "http://127.0.0.1:8001"
	CurrentRequestId = "1"
	CurrentChainId   = "17000"
	CurrentChain     = "ethereum"
)

func setupDb() *database.DB {
	dbConfig := config.DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Name:     "multichain",
		User:     "postgres",
		Password: "postgres",
	}

	newDB, _ := database.NewDB(context.Background(), dbConfig)
	return newDB
}
