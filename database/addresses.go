package database

import (
	"strings"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Addresses struct {
	GUID        uuid.UUID      `gorm:"primary_key" json:"guid"`
	Address     common.Address `gorm:"type:varchar;unique;not null" json:"address"`
	AddressType AddressType    `gorm:"type:varchar(10);not null; default:'eoa'" json:"address_type"`
	PublicKey   string         `gorm:"type:varchar;not null" json:"public_key"`
	Timestamp   uint64         `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
}

type AddressesView interface {
	AddressExists(requestId string, address *common.Address) (bool, AddressType)
	QueryAddressByToAddress(requestId string, toAddress *common.Address) (*Addresses, error)
	QueryHotWalletInfo(requestId string) (*Addresses, error)
	QueryColdWalletInfo(requestId string) (*Addresses, error)
	GetAllAddresses(requestId string) ([]*Addresses, error)
}

type AddressesDB interface {
	AddressesView

	StoreAddresses(requestId string, address []*Addresses) error
}

type addressesDB struct {
	gorm *gorm.DB
}

func NewAddressesDB(db *gorm.DB) AddressesDB {
	return &addressesDB{gorm: db}
}

func (a *addressesDB) AddressExists(requestId string, address *common.Address) (bool, AddressType) {
	var addressEntry Addresses
	err := a.gorm.Table(TableAddressesPrefix+requestId).
		Where("address = ?", strings.ToLower(address.String())).
		First(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, AddressTypeEOA
		}
		return false, AddressTypeEOA
	}
	return true, addressEntry.AddressType
}

func (a addressesDB) QueryAddressByToAddress(requestId string, toAddress *common.Address) (*Addresses, error) {
	var addressEntry Addresses
	err := a.gorm.Table(TableAddressesPrefix+requestId).
		Where("address = ?", strings.ToLower(toAddress.String())).
		First(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (a addressesDB) QueryHotWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := a.gorm.Table(TableAddressesPrefix+requestId).
		Where("address_type = ?", AddressTypeHot).
		First(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (a addressesDB) QueryColdWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := a.gorm.Table(TableAddressesPrefix+requestId).
		Where("address_type = ?", AddressTypeCold).
		First(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (a addressesDB) GetAllAddresses(requestId string) ([]*Addresses, error) {
	var addresses []*Addresses
	err := a.gorm.Table(TableAddressesPrefix + requestId).
		Find(&addresses).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return addresses, nil
}

func (a addressesDB) StoreAddresses(requestId string, addressList []*Addresses) error {

	for _, address := range addressList {
		address.Address = common.HexToAddress(address.Address.Hex())
	}
	return a.gorm.CreateInBatches(&addressList, len(addressList)).Error
}

func (a Addresses) validate() error {
	if a.Address == (common.Address{}) {
		return errors.New("address is required")
	}
	if a.PublicKey == "" {
		return errors.New("public_key is required")
	}
	if a.Timestamp <= 0 {
		return errors.New("invalid timestamp")
	}
	switch a.AddressType {
	case AddressTypeEOA, AddressTypeCold, AddressTypeHot:
		return nil
	default:
		return errors.New("invalid address type")
	}
}
