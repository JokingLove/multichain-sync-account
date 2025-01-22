package database

import (
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

type CreateTableDB interface {
	CreateTable(tableName, realTableName string)
}

type createTableDB struct {
	gorm *gorm.DB
}

func NewCreateTableDB(db *gorm.DB) CreateTableDB {
	return &createTableDB{gorm: db}
}

func (c *createTableDB) CreateTable(tableName, realTableName string) {
	err := c.gorm.Exec("CREATE TABLE IF NOT EXISTS " + tableName + " ( like " + realTableName + " including all )").Error
	if err != nil {
		log.Error("create table from base table fail", "err", err)
	}
}
