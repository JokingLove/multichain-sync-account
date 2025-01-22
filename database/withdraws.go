package database

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"math/big"
)

type Withdraws struct {
	// 基础信息
	GUID      uuid.UUID `gorm:"primaryKey;not null" json:"guid"`
	Timestamp uint64    `gorm:"not null" json:"timestamp"`
	Status    TxStatus  `gorm:"not null" json:"status"`

	// 区块信息
	BlockHash   common.Hash     `gorm:"column:block_hash;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int        `gorm:"column:block_number;serializer:u256" json:"block_number"`
	TxHash      common.Hash     `gorm:"column:tx_hash;serializer:bytes" json:"tx_hash"`
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
	TokenType    TokenType      `gorm:"column:token_type" json:"token_type"`
	TokenAddress common.Address `gorm:"column:token_address;serializer:bytes" json:"token_address"`
	TokenId      string         `json:"token_id"`
	TokenMeta    string         `json:"token_meta"`

	// 交易签名
	TxSignHex string `gorm:"column:tx_sign_hex" json:"tx_sign_hex"`
}

type WithdrawView interface {
	QueryNotifyWithdraws(requestId string) ([]*Withdraws, error)
	QueryWithdrawsByHash(requestId string, txHash common.Hash) (*Withdraws, error)
	QueryWithdrawsById(requestId string, guid string) (*Withdraws, error)
	UnSendWithdrawList(requestId string) ([]*Withdraws, error)
}

type WithdrawDB interface {
	WithdrawView

	StoreWithdraw(requestId string, withdraw *Withdraws) error
	UpdateWithdrawByTxHash(requestId string, txHash common.Hash, signedTx string, status TxStatus) error
	UpdateWithdrawById(requestId string, guid string, signedTx string, status TxStatus) error
	UpdateWithdrawStatusById(requestId string, status TxStatus, withdrawList []*Withdraws) error
	UpdateWithdrawStatusByTxHash(requestId string, status TxStatus, withdrawList []*Withdraws) error
	UpdateWithdrawListByTxHash(requestId string, withdrawList []*Withdraws) error
	UpdateWithdrawListById(requestId string, withdrawList []*Withdraws) error
}

type withdrawDB struct {
	gorm *gorm.DB
}

func NewWithdrawDB(db *gorm.DB) WithdrawDB {
	return &withdrawDB{
		gorm: db,
	}
}

func (db withdrawDB) StoreWithdraw(requestId string, withdraw *Withdraws) error {
	result := db.gorm.Table(TableWithdrawsPrefix + requestId).Create(withdraw)
	return result.Error
}

func (db withdrawDB) QueryNotifyWithdraws(requestId string) ([]*Withdraws, error) {
	var notifyWithdraws []*Withdraws
	result := db.gorm.Table(TableWithdrawsPrefix+requestId).
		Where("status = ? or status = ?", TxStatusWalletDone, TxStatusNotified).
		Find(&notifyWithdraws)
	if result.Error != nil {
		return nil, fmt.Errorf("query notify withdraws failed: %v", result.Error)
	}

	return notifyWithdraws, nil
}

func (db withdrawDB) QueryWithdrawsByHash(requestId string, txHash common.Hash) (*Withdraws, error) {
	var withdraws Withdraws
	result := db.gorm.Table(TableWithdrawsPrefix+requestId).
		Where("tx_hash = ?", txHash.String()).
		Take(&withdraws)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &withdraws, nil
}

func (db withdrawDB) QueryWithdrawsById(requestId string, guid string) (*Withdraws, error) {
	var withdraws Withdraws
	result := db.gorm.Table(TableWithdrawsPrefix+requestId).
		Where("guid = ?", guid).
		Take(&withdraws)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &withdraws, nil
}

func (db withdrawDB) UnSendWithdrawList(requestId string) ([]*Withdraws, error) {
	var withdrawList []*Withdraws
	result := db.gorm.Table(TableWithdrawsPrefix+requestId).
		Where("status = ?", TxStatusCreateUnsigned).
		Find(&withdrawList)
	if result.Error != nil {
		return nil, fmt.Errorf("query unsign withdraws failed: %v", result.Error)
	}
	return withdrawList, nil
}

func (db withdrawDB) UpdateWithdrawByTxHash(requestId string, txHash common.Hash, signedTx string, status TxStatus) error {
	tableName := fmt.Sprintf("%s_%s", TableWithdrawsPrefix, requestId)

	if err := db.CheckWithdrawExistsByTxHash(tableName, txHash); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	// 3.执行更新
	if err := db.gorm.Table(tableName).
		Where("tx_hash = ?", txHash.String()).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("update withdraws failed: %v", err)
	}

	// 4. 记录日志
	log.Info(
		"Update withdraw success",
		"requestId", requestId,
		"txHash", txHash.String(),
		"status", status,
		"updates", updates,
	)
	return nil
}

func (db withdrawDB) UpdateWithdrawById(requestId string, guid string, signedTx string, status TxStatus) error {
	tableName := fmt.Sprintf("%s_%s", TableWithdrawsPrefix, requestId)

	if err := db.CheckWithdrawExistsById(tableName, guid); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	// 3.执行更新
	if err := db.gorm.Table(tableName).
		Where("guid = ?", guid).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("update withdraws failed: %v", err)
	}

	// 4. 记录日志
	log.Info(
		"Update withdraw success",
		"requestId", requestId,
		"guid", guid,
		"status", status,
		"updates", updates,
	)
	return nil
}

func (db withdrawDB) UpdateWithdrawStatusById(requestId string, status TxStatus, withdrawList []*Withdraws) error {
	if len(withdrawList) == 0 {
		return nil
	}

	tableName := fmt.Sprintf("%s_%s", TableWithdrawsPrefix, requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var guids []uuid.UUID
		for _, withdraw := range withdrawList {
			guids = append(guids, withdraw.GUID)
		}

		result := tx.Table(tableName).
			Where("guid IN (?)", guids).
			Update("status", status)
		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn(
				"No withdraws updated",
				"requestId", requestId,
				"expectedCount", len(withdrawList),
			)
		}
		log.Info("Batch update withdraws status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}

func (db withdrawDB) UpdateWithdrawStatusByTxHash(requestId string, status TxStatus, withdrawList []*Withdraws) error {
	if len(withdrawList) == 0 {
		return nil
	}

	tableName := fmt.Sprintf("%s_%s", TableWithdrawsPrefix, requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var txHashList []common.Hash
		for _, withdraw := range withdrawList {
			txHashList = append(txHashList, withdraw.TxHash)
		}

		result := tx.Table(tableName).
			Where("tx_hash IN (?)", txHashList).
			Update("status", status)
		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn(
				"No withdraws updated",
				"requestId", requestId,
				"expectedCount", len(withdrawList),
			)
		}
		log.Info("Batch update withdraws status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}

func (db withdrawDB) UpdateWithdrawListByTxHash(requestId string, withdrawList []*Withdraws) error {
	if len(withdrawList) == 0 {
		return nil
	}

	tableName := fmt.Sprintf("%s_%s", TableWithdrawsPrefix, requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, withdraw := range withdrawList {
			// update each record individually based on TxHash
			result := tx.Table(tableName).
				Where("tx_hash = ?", withdraw.TxHash.String()).
				Updates(map[string]interface{}{
					"status": withdraw.Status,
					"amount": withdraw.Amount,
				})

			// check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash %s: %w", withdraw.TxHash.Hex(), result.Error)
			}

			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				fmt.Errorf("No withdraws updated for TxHash %s\n", withdraw.TxHash.Hex())
			} else {
				// Log success message with the number of rows affected
				fmt.Printf("Update withdraw for TxHash : %s, status: %s, amount: %d\n", withdraw.TxHash.Hex(), withdraw.Status, withdraw.Amount)
			}

		}
		return nil
	})
}

func (db withdrawDB) UpdateWithdrawListById(requestId string, withdrawList []*Withdraws) error {
	if len(withdrawList) == 0 {
		return nil
	}

	tableName := fmt.Sprintf("%s_%s", TableWithdrawsPrefix, requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, withdraw := range withdrawList {
			result := db.gorm.Table(tableName).
				Where("guid = ?", withdraw.GUID.String()).
				Updates(map[string]interface{}{
					"status":  withdraw.Status,
					"amount":  withdraw.Amount,
					"tx_hash": withdraw.TxHash.String(),
				})
			// Check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash %s: %w", withdraw.TxHash.Hex(), result.Error)
			}

			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				fmt.Printf("No withdraws updated for TxHash: %s\n", withdraw.TxHash.Hex())
			} else {
				// Log success message with the number of rows affected
				fmt.Printf("Updated withdraw for TxHash: %s, status: %s, amount: %s\n", withdraw.TxHash.Hex(), withdraw.Status, withdraw.Amount.String())
			}
		}
		return nil
	})
}

func (db withdrawDB) CheckWithdrawExistsByTxHash(tableName string, hash common.Hash) error {
	var exist bool
	err := db.gorm.Table(tableName).
		Where("tx_hash = ?", hash.String()).
		Select("1").
		Find(&exist).Error
	if err != nil {
		return fmt.Errorf("check withdraw exist failed: %v", err)
	}

	if !exist {
		return fmt.Errorf("withdraw not found: %s", hash.String())
	}
	return nil
}

func (db withdrawDB) CheckWithdrawExistsById(tableName string, guid string) error {
	var exist bool
	err := db.gorm.Table(tableName).
		Where("guid = ?", guid).
		Select("1").
		Find(&exist).Error
	if err != nil {
		return fmt.Errorf("check withdraw exist failed: %v", err)
	}

	if !exist {
		return fmt.Errorf("withdraw not found: %s", guid)
	}
	return nil
}
