package database

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"os"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/JokingLove/multichain-sync-account/common/retry"
	"github.com/JokingLove/multichain-sync-account/config"
)

type DB struct {
	gorm *gorm.DB

	CreateTable CreateTableDB
	Blocks      BlocksDB
	Addresses   AddressesDB
	Balances    BalancesDB
	Deposits    DepositsDB
	Tokens      TokensDB
	Business    BusinessDB
	Trasactions TransactionsDB
	//Internals InternalsDB
	Withdraws WithdrawDB
}

func NewDB(ctx context.Context, dbConfig config.DBConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Name)
	if dbConfig.Port != 0 {
		dsn += fmt.Sprintf(" port=%d", dbConfig.Port)
	}

	if dbConfig.User != "" {
		dsn += fmt.Sprintf(" user=%s", dbConfig.User)
	}

	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}

	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
		CreateBatchSize:        3_000,
		Logger:                 newLogger,
	}

	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	gormDbBox, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gormDb, err := gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		return gormDb, err
	})

	if err != nil {
		return nil, err
	}

	db := &DB{
		gorm:        gormDbBox,
		CreateTable: NewCreateTableDB(gormDbBox),
		Blocks:      NewBlocksDB(gormDbBox),
		Addresses:   NewAddressesDB(gormDbBox),
		Balances:    NewBalancesDB(gormDbBox),
		Deposits:    NewDepositsDB(gormDbBox),
		Tokens:      NewTokensDB(gormDbBox),
		Business:    NewBusinessDB(gormDbBox),
		Withdraws:   NewWithdrawDB(gormDbBox),
		// TODO
	}

	return db, nil
}

func (db *DB) Transaction(fn func(db *DB) error) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		txDB := &DB{
			gorm: tx,
		}
		return fn(txDB)
	})

}

func (db *DB) Close() error {
	sql, err := db.gorm.DB()
	if err != nil {
		return err
	}
	return sql.Close()
}

func (db *DB) ExecuteSQLMigration(migrationsFolder string) error {
	err := filepath.Walk(migrationsFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to process migration file: %s", path))
		}
		if info.IsDir() {
			return nil
		}
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to read file: %s", path))
		}

		execErr := db.gorm.Exec(string(fileContent)).Error
		if execErr != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to execute sql: %s", path))
		}
		return nil
	})
	return err
}
