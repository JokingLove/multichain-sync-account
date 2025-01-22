package database

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Business struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BusinessUid string    `json:"business_uid"`
	NotifyUrl   string    `json:"notify_url"`
	Timestamp   uint64    `json:"timestamp"`
}

type BusinessView interface {
	QueryBusinessList() ([]*Business, error)
	QueryBusinessByUuid(string) (*Business, error)
}

type BusinessDB interface {
	BusinessView

	StoreBusiness(*Business) error
}

type businessDB struct {
	gorm *gorm.DB
}

func NewBusinessDB(db *gorm.DB) BusinessDB {
	return &businessDB{gorm: db}
}

func (db businessDB) QueryBusinessList() ([]*Business, error) {
	var businesses []*Business
	err := db.gorm.Table("business").Find(&businesses).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return businesses, nil
}

func (db businessDB) QueryBusinessByUuid(uid string) (*Business, error) {
	var business *Business
	result := db.gorm.Table("business").Where("business_uid = ?", uid).
		First(&business)
	if result.Error != nil {
		log.Error("query business by uid failed", "err", result.Error)
		return nil, result.Error
	}
	return business, nil
}

func (db businessDB) StoreBusiness(business *Business) error {
	result := db.gorm.Table("business").Create(business)
	return result.Error
}
