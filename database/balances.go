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

func (db balanceDB) UpdateOrCreate(s string, balances []*TokenBalance) error {
	//TODO implement me
	panic("implement me")
}

func (db balanceDB) StoreBalances(s string, balances []*Balances) error {
	//TODO implement me
	panic("implement me")
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
