package database

import (
	"github.com/JokingLove/multichain-sync-account/rpcclient/chain-account/account"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"math/big"
)

type Transactions struct {
	GUID         uuid.UUID        `gorm:"primaryKey;type:uuid"`
	BlockHash    common.Hash      `gorm:"serializer:bytes;column:block_hash" json:"block_hash"`
	BlockNumber  *big.Int         `gorm:"serializer:uint256" json:"block_number"`
	Hash         common.Hash      `gorm:"serializer:bytes" json:"hash"`
	FromAddress  common.Address   `gorm:"serializer:bytes" json:"from_address"`
	ToAddress    common.Address   `gorm:"serializer:bytes" json:"to_address"`
	TokenAddress common.Address   `gorm:"serializer:bytes" json:"token_address"`
	TokenId      string           `gorm:"column:token_id" json:"token_id"`
	TokenMeta    string           `gorm:"column:token_meta" json:"token_meta"`
	Fee          *big.Int         `gorm:"serializer:uint256" json:"fee"`
	Amount       *big.Int         `gorm:"serializer:uint256" json:"amount"`
	Status       account.TxStatus `json:"status"`
	TxType       TransactionType  `json:"tx_type"`
	Timestamp    uint64           `json:"timestamp"`
}

type TransactionsView interface {
	QueryTransactionByHash(requestId string, hash common.Hash) (*Transactions, error)
}

type TransactionsDB interface {
	TransactionsView

	StoreTransactions(string, []*Transactions, uint64) error
	UpdateTransactionsStatus(requestId string, blockNumber *big.Int) error
	UpdateTransactionStatus(requestId string, txList []*Transactions) error
}

type transactionsDB struct {
	gorm *gorm.DB
}

func NewTransactionsDB(db *gorm.DB) TransactionsDB {
	return &transactionsDB{gorm: db}
}

func (db transactionsDB) QueryTransactionByHash(requestId string, hash common.Hash) (*Transactions, error) {
	var transactions Transactions
	result := db.gorm.Table(TableTransactionsPrefix+requestId).
		Where("hash = ?", hash).
		Take(&transactions)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &transactions, nil
}

func (db transactionsDB) StoreTransactions(requestId string, transactions []*Transactions, num uint64) error {
	result := db.gorm.Table(TableTransactionsPrefix+requestId).
		CreateInBatches(transactions, len(transactions))
	return result.Error
}

func (db transactionsDB) UpdateTransactionsStatus(requestId string, blockNumber *big.Int) error {
	result := db.gorm.Table(TableTransactionsPrefix+requestId).
		Where("block_number = ? and status = ?", blockNumber, 0).
		Updates(map[string]interface{}{
			"status": gorm.Expr("GREATEST(1)"),
		})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	return nil
}

func (db transactionsDB) UpdateTransactionStatus(requestId string, txList []*Transactions) error {
	if len(txList) == 0 {
		return nil
	}

	for i := 0; i < len(txList); i++ {
		var transactionSingle = Transactions{}
		result := db.gorm.Table(TableTransactionsPrefix + requestId).
			Where(&Transactions{Hash: txList[i].Hash}).Take(&transactionSingle)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			return result.Error
		}
		transactionSingle.Status = txList[i].Status
		transactionSingle.Fee = txList[i].Fee
		err := db.gorm.Table(TableTransactionsPrefix + requestId).Save(&transactionSingle).Error
		if err != nil {
			return err
		}
	}
	return nil
}
