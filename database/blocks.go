package database

import (
	"math/big"

	"github.com/JokingLove/multichain-sync-account/rpcclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func BlockHeaderFromHeader(header *types.Header) rpcclient.BlockHeader {
	return rpcclient.BlockHeader{
		Hash:       header.Hash(),
		ParentHash: header.ParentHash,
		Number:     header.Number,
		Timestamp:  header.Time,
	}
}

type Blocks struct {
	Hash       common.Hash `gorm:"primaryKey; serializer:bytes" json:"hash"`
	ParentHash common.Hash `gorm:"serializer:bytes" json:"parent_hash"`
	Number     *big.Int    `gorm:"serializer:u256" json:"number"`
	Timestamp  uint64
}

type BlocksView interface {
	LatestBlocks() (*rpcclient.BlockHeader, error)
}

type BlocksDB interface {
	BlocksView

	StoreBlocks([]Blocks) error
}

type blocksDB struct {
	gorm *gorm.DB
}

func NewBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{
		gorm: db,
	}
}

func (b blocksDB) LatestBlocks() (*rpcclient.BlockHeader, error) {
	var header Blocks
	result := b.gorm.Order("number desc").Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return (*rpcclient.BlockHeader)(&header), nil
}

func (b blocksDB) StoreBlocks(blocks []Blocks) error {
	result := b.gorm.CreateInBatches(&blocks, len(blocks))
	return result.Error
}
