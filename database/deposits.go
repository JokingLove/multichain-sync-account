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

type Deposits struct {
	GUID      uuid.UUID `gorm:"primaryKey"`
	Timestamp uint64    `gorm:"not null; check: timestamp > 0" json:"timestamp"`
	Status    TxStatus  `gorm:"type:varchar(10);not null" json:"status"`
	Confirms  uint8     `gorm:"not null; default 0" json:"confirms"`

	BlockHash   common.Hash     `gorm:"type:varchar;not null;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int        `gorm:"not null; check: block_number > 0; serializer: u256" json:"block_number"`
	TxHash      common.Hash     `gorm:"type:varchar;not null;serializer:bytes" json:"tx_hash"`
	TxType      TransactionType `gorm:"type:varchar;not null" json:"tx_type"`

	FromAddress common.Address `gorm:"type:varchar;not null;serializer:bytes" json:"from_address"`
	ToAddress   common.Address `gorm:"type:varchar;not null;serializer:bytes" json:"to_address"`
	Amount      *big.Int       `gorm:"not null;serializer:u256" json:"amount"`

	GasLimit          uint64 `gorm:"not null" json:"gas_limit"`
	MaxFeePerGas      string `gorm:"type:varchar;not null" json:"max_fee_per_gas"`
	MaxPriorityFeeGas string `gorm:"type:varchar;not null" json:"max_priority_fee_gas"`

	TokenType    TokenType      `gorm:"type:varchar;not null" json:"token_type"`
	TokenAddress common.Address `gorm:"type:varchar;not null" json:"token_address"`
	TokenId      string         `gorm:"type:varchar;not null" json:"token_id"`
	TokenMeta    string         `gorm:"type:varchar;not null" json:"token_meta"`

	TxSignHex string `gorm:"type:varchar;not null" json:"tx_sign_hex"`
}

type DepositsView interface {
	QueryNotifyDeposits(requestId string) ([]*Deposits, error)
	QueryDepositsByTxHash(requestId string, txHash common.Hash) (*Deposits, error)
	QueryDepositsById(requestId string, guid string) (*Deposits, error)
}

type DepositsDB interface {
	DepositsView

	StoreDeposits(string, []*Deposits) error
	UpdateDepositsConfirms(requestId string, blockNumber uint64, confirms uint64) error
	UpdateDepositById(requestId string, guid string, signedTx string, status TxStatus) error
	UpdateDepositsStatusById(requestId string, status TxStatus, depositsList []*Deposits) error
	UpdateDepositsStatusByTxHash(requestId string, status TxStatus, depositsList []*Deposits) error
	UpdateDepositListByTxHash(requestId string, depositsList []*Deposits) error
	UpdateDepositListById(requestId string, depositsList []*Deposits) error
}

type depositsDB struct {
	gorm *gorm.DB
}

func NewDepositsDB(db *gorm.DB) DepositsDB {
	return &depositsDB{
		gorm: db,
	}
}

func (db depositsDB) QueryNotifyDeposits(requestId string) ([]*Deposits, error) {
	var deposits []*Deposits
	result := db.gorm.Table(TableDepositsPrefix+requestId).
		Where("status = ? or status = ? ", TxStatusWalletDone, TxStatusNotified).
		Find(deposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}
	return deposits, nil
}

func (db depositsDB) QueryDepositsByTxHash(requestId string, txHash common.Hash) (*Deposits, error) {
	var deposits *Deposits
	result := db.gorm.Table(TableDepositsPrefix+requestId).
		Where("tx_hash = ?", txHash.String()).
		Take(&deposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return deposits, nil
}

func (db depositsDB) QueryDepositsById(requestId string, guid string) (*Deposits, error) {
	var deposits *Deposits
	result := db.gorm.Table(TableDepositsPrefix+requestId).
		Where("guid = ?", guid).
		Take(&deposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return deposits, nil
}

func (db depositsDB) StoreDeposits(requestId string, deposits []*Deposits) error {
	if len(deposits) == 0 {
		return nil
	}
	result := db.gorm.Table(TableDepositsPrefix+requestId).
		CreateInBatches(deposits, len(deposits))
	if result.Error != nil {
		log.Error("create Deposits batch failed", "err", result.Error)
	}
	return result.Error
}

// 查询所有还没有过确认位的交易，用最新的区块减去对应区块更新确认，如果这个大于我们预设的确认位，那么这笔交易可以认为已经入账
func (db depositsDB) UpdateDepositsConfirms(requestId string, blockNumber uint64, confirms uint64) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var unConfirmDeposits []*Deposits
		result := tx.Table(TableDepositsPrefix+requestId).
			Where("block_number <= ? and status = ?", blockNumber, TxStatusBoradcasted).
			Find(&unConfirmDeposits)
		if result.Error != nil {
			return result.Error
		}

		for _, deposit := range unConfirmDeposits {
			chainConfirm := blockNumber - deposit.BlockNumber.Uint64()
			if chainConfirm >= confirms {
				deposit.Confirms = uint8(confirms)
				deposit.Status = TxStatusWalletDone
			} else {
				deposit.Confirms = uint8(chainConfirm)
			}

			if err := tx.Table(TableDepositsPrefix + requestId).Save(deposit).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (db depositsDB) UpdateDepositById(requestId string, guid string, signedTx string, status TxStatus) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var deposits Deposits
		result := tx.Table(TableDepositsPrefix+requestId).
			Where("guid = ?", guid).
			Take(&deposits)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("deposits not found for GUID: %s", guid)
			}
			return result.Error
		}

		deposits.Status = status
		deposits.TxSignHex = signedTx

		if err := tx.Table(TableDepositsPrefix + requestId).Save(&deposits).Error; err != nil {
			return fmt.Errorf("failed to update deposit for GUID: %s, error: %w", guid, err)
		}
		return nil
	})
}

func (db depositsDB) UpdateDepositsStatusById(requestId string, status TxStatus, depositsList []*Deposits) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, deposit := range depositsList {
			var depositSingle Deposits
			result := tx.Table(TableDepositsPrefix+requestId).
				Where("guid = ?", deposit.GUID.String()).Take(&depositSingle)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					continue
				}
				return result.Error
			}

			depositSingle.Status = status
			if err := tx.Table(TableDepositsPrefix + requestId).Save(&depositSingle).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (db depositsDB) UpdateDepositsStatusByTxHash(requestId string, status TxStatus, depositsList []*Deposits) error {
	if len(depositsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("%s_%s", TableDepositsPrefix, requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var txHashList []string
		for _, deposit := range depositsList {
			txHashList = append(txHashList, deposit.TxHash.String())
		}
		result := tx.Table(tableName).
			Where("hash IN ?", txHashList).
			Update("status", status)
		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No deposits updated", "requestId", requestId, "expectedCount", len(txHashList))
		}
		log.Info("Batch update deposits status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}

func (db depositsDB) UpdateDepositListByTxHash(requestId string, depositsList []*Deposits) error {
	if len(depositsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("%s_%s", TableDepositsPrefix, requestId)
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, deposit := range depositsList {
			result := tx.Table(tableName).
				Where("hash = ?", deposit.TxHash.String()).
				Updates(map[string]interface{}{
					"status": deposit.Status,
					"amount": deposit.Amount,
				})

			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash: %s, error: %w", deposit.TxHash.String(), result.Error)
			}
			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				fmt.Printf("No deposits updated for TxHash: %s\n", deposit.TxHash.Hex())
			} else {
				// Log success message with the number of rows affected
				fmt.Printf("Updated deposit for TxHash: %s, status: %s, amount: %s\n", deposit.TxHash.Hex(), deposit.Status, deposit.Amount.String())
			}
		}

		return nil
	})
}

func (db depositsDB) UpdateDepositListById(requestId string, depositsList []*Deposits) error {
	if len(depositsList) == 0 {
		return nil
	}

	tableName := fmt.Sprintf("%s_%s", TableDepositsPrefix, requestId)
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, deposit := range depositsList {
			result := tx.Table(tableName).
				Where("guid = ?", deposit.GUID.String()).
				Updates(map[string]interface{}{
					"status": deposit.Status,
					"amount": deposit.Amount,
					"hash":   deposit.TxHash.String(),
				})
			// Check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for GUID %s: %w", deposit.GUID.String(), result.Error)
			}

			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				fmt.Printf("No deposits updated for GUID: %s\n", deposit.GUID.String())
			} else {
				// Log success message with the number of rows affected
				fmt.Printf("Updated deposit for GUID: %s, status: %s, amount: %s\n", deposit.GUID.String(), deposit.Status, deposit.Amount.String())
			}
		}
		return nil
	})
}
