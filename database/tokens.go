package database

import (
	"errors"
	"math/big"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tokens struct {
	GUID          uuid.UUID      `gorm:"primaryKey" db:"guid"`
	TokenAddress  common.Address `gorm:"serializer:bytes" json:"token_address"`
	Decimals      uint8          `json:"decimals"`
	TokenName     string         `json:"token_name"`
	CollectAmount *big.Int       `gorm:"serializer:u256" json:"collect_amount"`
	ColdAmount    *big.Int       `gorm:"serializer:u256" json:"cold_amount"`
	TimeStamp     uint64         `json:"time_stamp"`
}

type TokensView interface {
	TokensInfoByAddress(string, string) (*Tokens, error)
}

type TokensDB interface {
	TokensView

	StoreTokens(string, []Tokens) error
}

type tokensDB struct {
	gorm *gorm.DB
}

func NewTokensDB(db *gorm.DB) TokensDB {
	return &tokensDB{gorm: db}
}

func (t tokensDB) TokensInfoByAddress(requestId string, tokenAddress string) (*Tokens, error) {
	var tokens Tokens
	err := t.gorm.Table(TableTokensPrefix+requestId).
		Where("token_address = ?", tokenAddress).
		First(&tokens).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tokens, nil
}

func (t tokensDB) StoreTokens(s string, tokensList []Tokens) error {
	result := t.gorm.Table(TableTokensPrefix+s).CreateInBatches(&tokensList, len(tokensList))
	return result.Error
}
