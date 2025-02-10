package database

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Internals struct {

	// 基础信息
	GUID      uuid.UUID `gorm:"primaryKey;not null" json:"guid"`
	Timestamp uint64    `gorm:"not null" json:"timestamp"`
	Status    TxStatus  `gorm:"not null" json:"status"`

	// 区块信息
	BlockHash   common.Hash     `gorm:"column:block_hash;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int        `gorm:"column:block_number;serializer:u256" json:"block_number"`
	TxHash      common.Hash     `gorm:"column:tx_hash;serializer:bytes" json:"hash"`
	TxType      TransactionType `gorm:"column:tx_type; not null" json:"tx_type"`

	// 交易基础信息
	FromAddress common.Address `gorm:"column:from_address;serializer:bytes" json:"from_address"`
	ToAddress   common.Address `gorm:"column:to_address;serializer:bytes" json:"to_address"`
	Amount      *big.Int       `gorm:"column:amount;serializer:u256" json:"amount"`

	// Gas 费用
	GasLimit             uint64 `gorm:"column:gas_limit" json:"gas_limit"`
	MaxFeePerGas         string `gorm:"column:max_fee_per_gas" json:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `gorm:"column:max_priority_fee_per_gas" json:"max_priority_fee_per_gas"`

	// Token 相关信息
	TokenType    TokenType      `gorm:"column:token_type" json:"token_type"` // ETH, ERC20, ERC721, ERC1155
	TokenAddress common.Address `gorm:"column:token_address;serializer:bytes" json:"token_address"`
	TokenId      string         `json:"token_id"`   // ERC721 / ERC1155 的 token id
	TokenMeta    string         `json:"token_meta"` // Token 元数据

	// 交易签名
	TxSignHex string `gorm:"column:tx_sign_hex" json:"tx_sign_hex"`
}

type InternalsView interface {
	QueryNotifyInternals(requestId string) ([]*Internals, error)
	QueryInternalByTxHash(requestId string, txHash common.Hash) (*Internals, error)
	QueryInternalById(requestId string, guid string) (*Internals, error)
	UnSendInternalList(requestId string) ([]*Internals, error)
}

type InternalsDB interface {
	InternalsView

	StoreInternal(string, *Internals) error
	UpdateInternalByTxHash(requestId string, txHash common.Hash, signedTx string, status TxStatus) error
	UpdateInternalById(requestId string, guid string, signedTx string, status TxStatus) error
	UpdateInternalStatusByTxHash(requestId string, status TxStatus, internalsList []*Internals) error
	UpdateInternalListByHash(requestId string, internalsList []*Internals) error
	UpdateInternalListById(requestId string, internalsList []*Internals) error
}

type internalsDB struct {
	gorm *gorm.DB
}

type GasInfo struct {
	GasLimit             uint64
	MaxFeePerGas         string
	MaxPriorityFeePerGas string
}

func NewInternalsDB(db *gorm.DB) InternalsDB {
	return &internalsDB{
		gorm: db,
	}
}

func (db internalsDB) QueryNotifyInternals(requestId string) ([]*Internals, error) {
	var notifyInternals []*Internals
	result := db.gorm.Table(TableInternalsPrefix+requestId).
		Where("status = ? or status = ?", TxStatusWalletDone, TxStatusNotified).
		Find(&notifyInternals)
	if result.Error != nil {
		return nil, result.Error
	}
	return notifyInternals, nil
}

func (db internalsDB) QueryInternalByTxHash(requestId string, txHash common.Hash) (*Internals, error) {
	var internals Internals
	result := db.gorm.Table(TableInternalsPrefix+requestId).
		Where("hash = ?", txHash.String()).
		Take(&internals)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &internals, nil
}

func (db internalsDB) QueryInternalById(requestId string, guid string) (*Internals, error) {
	var internals Internals
	result := db.gorm.Table(TableInternalsPrefix+requestId).
		Where("guid = ?", guid).
		Take(&internals)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &internals, nil
}

func (db internalsDB) UnSendInternalList(requestId string) ([]*Internals, error) {
	var internals []*Internals
	result := db.gorm.Table(TableInternalsPrefix+requestId).
		Where("status = ?", TxStatusSigned).
		Find(&internals)
	if result.Error != nil {
		return nil, result.Error
	}
	return internals, nil
}

func (db internalsDB) StoreInternal(requestId string, internals *Internals) error {
	return db.gorm.Create(internals).Error
}

func (db internalsDB) UpdateInternalByTxHash(requestId string, txHash common.Hash, signedTx string, status TxStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table(TableInternalsPrefix+requestId).
		Where("hash = ?", txHash.String()).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (db internalsDB) UpdateInternalById(requestId string, guid string, signedTx string, status TxStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table(TableInternalsPrefix+requestId).
		Where("guid  = ?", guid).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (db internalsDB) UpdateInternalStatusByTxHash(requestId string, status TxStatus, internalsList []*Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := TableInternalsPrefix + requestId

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var txHashList []string
		for _, internals := range internalsList {
			txHashList = append(txHashList, internals.TxHash.String())
		}

		result := tx.Table(tableName).
			Where("hash IN (?)", txHashList).
			Update("status", status)

		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No internals updated",
				"requestId", requestId,
				"expectedCount", len(internalsList),
			)
		}
		log.Info("Batch update internals status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)
		return nil
	})
}

func (db internalsDB) UpdateInternalListByHash(requestId string, internalsList []*Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := TableInternalsPrefix + requestId

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, internals := range internalsList {
			// update each record individually base on txhash
			result := tx.Table(tableName).
				Where("hash = ?", internals.TxHash.String()).
				Updates(map[string]interface{}{
					"status": internals.Status,
					"amount": internals.Amount,
				})
			// check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash  %s : %w", internals.TxHash, result.Error)
			}
			// log a warning if no rows were updated
			if result.RowsAffected == 0 {
				log.Info("No internals updated for TxHash", "txhash", internals.TxHash.String())
			} else {
				// log success message with the number of rows affected
				log.Info("update internals for ", "txHash", internals.TxHash.String(), "amount", internals.Amount)
			}
		}
		return nil
	})
}

func (db internalsDB) UpdateInternalListById(requestId string, internalsList []*Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := TableInternalsPrefix + requestId

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, internals := range internalsList {
			result := tx.Table(tableName).
				Where("guid = ?", internals.GUID.String()).
				Updates(map[string]interface{}{
					"status": internals.Status,
					"amount": internals.Amount,
				})

			if result.Error != nil {
				return fmt.Errorf("update failed by id   %s : %w", internals.GUID.String(), result.Error)
			}

			if result.RowsAffected == 0 {
				log.Info("No internals updated by id", "id", internals.GUID.String())
			} else {
				log.Info("update internals for ", "id", internals.GUID.String(), "amount", internals.Amount)
			}
		}
		return nil
	})
}
