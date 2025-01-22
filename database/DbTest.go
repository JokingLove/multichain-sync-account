package database

import (
	"context"
	"github.com/JokingLove/multichain-sync-account/config"
)

func SetupDb() *DB {
	dbConfig := config.DbConfigTest()

	newDB, _ := NewDB(context.Background(), *dbConfig)

	return newDB
}
