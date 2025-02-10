package database

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Balances struct {
	GUID         uuid.UUID      `gorm:"primary_key" json:"guid"`
	Address      common.Address `gorm:"serializer:bytes;"json:"address"`
	TokenAddress common.Address `gorm:"serializer:bytes;"json:"token_address"`
	AddressType  AddressType    `gorm:"type:varchar(10);not null;"json:"address_type"`
	Balance      *big.Int       `gorm:"not null;default:0;"json:"balance"`
	LockBalance  *big.Int       `gorm:"not null;default:0;"json:"lock_balance"`
	Timestamp    uint64         `gorm:"not null;"json:"timestamp"`
}

type BalancesView interface {
	QueryWalletBalanceByTokenAndAddress(
		requestId string,
		addressType AddressType,
		address, tokenAddress common.Address,
	) (*Balances, error)
}

type BalancesDB interface {
	BalancesView

	UpdateOrCreate(string, []*TokenBalance) error
	StoreBalances(string, []*Balances) error
	UpdateBalanceListByTwoAddress(string, []*Balances) error
	UpdateBalance(string, *Balances) error
}

type balanceDB struct {
	gorm *gorm.DB
}

func NewBalancesDB(db *gorm.DB) BalancesDB {
	return &balanceDB{gorm: db}
}

func (db balanceDB) QueryWalletBalanceByTokenAndAddress(requestId string, addressType AddressType, address, tokenAddress common.Address) (*Balances, error) {
	balance, err := db.queryBalance(requestId, address, tokenAddress)
	if err == nil {
		return balance, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.createInitialBalance(requestId, addressType, address, tokenAddress)
	}
	return nil, fmt.Errorf("query balance failed: %w", err)
}

func (db balanceDB) UpdateOrCreate(requestId string, balances []*TokenBalance) error {
	if len(balances) == 0 {
		return nil
	}
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balances {
			log.Info("Processing balance update ",
				"txType", balance.TxType,
				"from", balance.FromAddress,
				"to", balance.ToAddress,
				"tokenAddress", balance.TokenAddress,
				"amount", balance.Balance,
			)

			if err := db.handleBalanceUpdate(tx, requestId, balance); err != nil {
				return fmt.Errorf("failed to handle balance update: %w", err)
			}
		}
		return nil
	})
}

func (db balanceDB) StoreBalances(requestId string, balances []*Balances) error {
	valueList := make([]Balances, len(balances))
	for i, balance := range balances {
		if balance != nil {
			balance.Address = common.HexToAddress(balance.Address.Hex())
			balance.TokenAddress = common.HexToAddress(balance.TokenAddress.Hex())
			valueList[i] = *balance
		}
	}
	return db.gorm.Table(TableBalancesPrefix+requestId).CreateInBatches(&valueList, len(valueList)).Error
}

func (db *balanceDB) UpdateAndSaveBalance(tx *gorm.DB, requestId string, balance *Balances) error {
	if balance == nil {
		return fmt.Errorf("balance can not be nil")
	}

	var currentBalance Balances
	result := tx.Table(TableBalancesPrefix+requestId).
		Where("address = ? and token_address = ?",
			balance.Address.String(), balance.TokenAddress.String()).
		Take(&currentBalance)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Debug("Balance record not found",
				"requestId", requestId,
				"address", balance.Address.String(),
				"tokenAddress", balance.TokenAddress.String(),
			)
			return nil
		}
		return fmt.Errorf("query balance failed: %w", result.Error)
	}

	currentBalance.Balance = balance.Balance // 上游修改这里不做重复计算
	currentBalance.LockBalance = new(big.Int).Add(currentBalance.LockBalance, balance.LockBalance)
	currentBalance.Timestamp = uint64(time.Now().Unix())

	if err := tx.Table(TableBalancesPrefix + requestId).Save(&currentBalance).Error; err != nil {
		log.Error("Failed to save balance",
			"requestId", requestId,
			"address", balance.Address.String(),
			"error", err)
		return fmt.Errorf("save balance failed: %w", err)
	}

	log.Debug("Balance updated and save successfully",
		"requestId", requestId,
		"address", balance.Address.String(),
		"tokenAddress", balance.TokenAddress.String(),
		"newBalance", currentBalance.Balance.String(),
		"lockBalance", currentBalance.LockBalance.String(),
	)
	return nil
}

func (db balanceDB) UpdateBalanceListByTwoAddress(requestId string, balanceList []*Balances) error {
	if len(balanceList) == 0 {
		return nil
	}

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balanceList {
			var currentBalance Balances
			result := tx.Table(TableBalancesPrefix+requestId).
				Where("address = ? and token_address = ?", balance.Address.String(), balance.TokenAddress.String()).
				Take(&currentBalance)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					continue
				}
				return fmt.Errorf("query balance failed: %w", result.Error)
			}

			currentBalance.Balance = new(big.Int).Sub(currentBalance.Balance, balance.LockBalance)
			currentBalance.LockBalance = balance.LockBalance
			currentBalance.Timestamp = balance.Timestamp

			if err := tx.Table(TableBalancesPrefix + requestId).Save(&currentBalance).Error; err != nil {
				return fmt.Errorf("save balance failed: %w", err)
			}
		}
		return nil
	})
}

func (db balanceDB) UpdateBalance(s string, balances *Balances) error {
	//TODO implement me
	panic("implement me")
}

func (db *balanceDB) queryBalance(
	requestId string,
	address, tokenAddress common.Address,
) (*Balances, error) {
	var balance Balances

	err := db.gorm.Table(TableBalancesPrefix+requestId).
		Where("address = ? and token_address = ?", strings.ToLower(address.String()),
			strings.ToLower(tokenAddress.String()),
		).Take(&balance).Error
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

func (db balanceDB) createInitialBalance(requestId string, addressType AddressType, address common.Address, tokenAddress common.Address) (*Balances, error) {
	balance := &Balances{
		GUID:         uuid.New(),
		Address:      address,
		TokenAddress: tokenAddress,
		AddressType:  addressType,
		Balance:      big.NewInt(0),
		LockBalance:  big.NewInt(0),
		Timestamp:    uint64(time.Now().Unix()),
	}

	if err := db.gorm.Table(TableBalancesPrefix + requestId).Create(balance).Error; err != nil {
		log.Error("failed to create initial balance", "requestId", requestId, "address", address, "tokenAddress", tokenAddress, "error", err)
		return nil, fmt.Errorf("create initial balance failed: %w", err)
	}

	log.Debug(""+
		"Created initial balance",
		"requestId", requestId, "address", address, "tokenAddress", tokenAddress,
	)
	return balance, nil
}

func (db balanceDB) handleBalanceUpdate(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	switch balance.TxType {
	case TxTypeDeposit:
		return db.handleDeposit(tx, requestId, balance)
	case TxTypeWithdraw:
		return db.handleWithdraw(tx, requestId, balance)
	case TxTypeCollection:
		return db.handleCollection(tx, requestId, balance)
	case TxTypeHot2Cold:
		return db.handleHot2Cold(tx, requestId, balance)
	case TxTypeCold2Hot:
		return db.handleCold2Hot(tx, requestId, balance)
	default:
		return fmt.Errorf("unsupported transaction type: %s", balance.TxType)
	}
}

func (db balanceDB) handleDeposit(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	userAddress, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeEOA, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user address failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}

	userAddress.Balance = new(big.Int).Add(userAddress.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, userAddress)
}

func (db balanceDB) handleWithdraw(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet address balance failed", "requestId", requestId, "address", balance.FromAddress, "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

func (db balanceDB) handleCollection(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	userWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeEOA, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("query user wallet address balance failed", "requestId", requestId, "address", balance.FromAddress, "err", err)
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("query hot wallet address balance failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}

	userWallet.Balance = new(big.Int).Sub(userWallet.Balance, balance.Balance)
	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)

	// update user wallet
	if err := db.UpdateAndSaveBalance(tx, requestId, userWallet); err != nil {
		log.Error("update user wallet failed for collection", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}
	// update hot wallet
	if err := db.UpdateAndSaveBalance(tx, requestId, hotWallet); err != nil {
		log.Error("update hot wallet failed for collection", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}
	return nil
}

func (db balanceDB) handleHot2Cold(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("hot2cold query hot wallet failed", "requestId", requestId, "address", balance.FromAddress, "err", err)
		return err
	}

	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeCold, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("hot2cold query cold wallet failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}

	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	coldWallet.Balance = new(big.Int).Add(coldWallet.Balance, balance.Balance)

	if err := db.UpdateAndSaveBalance(tx, requestId, hotWallet); err != nil {
		log.Error("hot2cold update hot wallet balance failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}

	if err := db.UpdateAndSaveBalance(tx, requestId, coldWallet); err != nil {
		log.Error("hot2cold update cold wallet balance failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}
	return nil
}

func (db balanceDB) handleCold2Hot(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("cold2hot query hot wallet failed", "requestId", requestId, "address", balance.FromAddress, "err", err)
		return err
	}

	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, AddressTypeCold, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("cold2hot query cold wallet failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}

	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	coldWallet.Balance = new(big.Int).Sub(coldWallet.Balance, balance.Balance)

	if err := db.UpdateAndSaveBalance(tx, requestId, hotWallet); err != nil {
		log.Error("cold2hot update hot wallet balance failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}

	if err := db.UpdateAndSaveBalance(tx, requestId, coldWallet); err != nil {
		log.Error("cold2hot update cold wallet balance failed", "requestId", requestId, "address", balance.ToAddress, "err", err)
		return err
	}
	return nil
}
