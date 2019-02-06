package db

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/notegio/openrelay/types"
)

// AssetProxy contains information about supported asset proxy.
type AssetProxy struct {
	ID              []byte         `gorm:"primary_key"`
	Name            string         `gorm:"index"`
	Address         *types.Address `gorm:"index"`
	ExchangeAddress *types.Address `gorm:"index"`
	ZeroEx          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Exchange        *Exchange `gorm:"foreignkey:exchange_address"`
}

// Save records the asset proxy in the database.
func (assetProxy *AssetProxy) Save(db *gorm.DB) *gorm.DB {
	log.Printf("Attempting to save asset proxy with ID %#x...", assetProxy.ID)
	result := db.Model(&AssetProxy{}).FirstOrCreate(assetProxy)
	if result.Error != nil {
		log.Printf("Unable to save asset proxy with ID %#x: %s", assetProxy.ID, result.Error.Error())
	}
	return result
}

// GetAssetProxyByID reads the asset proxy by it's ID from the database.
func GetAssetProxyByID(db *gorm.DB, id []byte) *AssetProxy {
	assetProxy := &AssetProxy{}
	if result := db.Where("id = ?", id).First(assetProxy); result.Error != nil {
		log.Printf("Unable to read asset proxy with ID %#x: %s", id, result.Error.Error())
		return nil
	}
	return assetProxy
}

// GetAssetProxyByName reads the asset proxy by it's name from the database.
func GetAssetProxyByName(db *gorm.DB, name string) *AssetProxy {
	assetProxy := &AssetProxy{}
	if result := db.Where("name = ?", name).First(assetProxy); result.Error != nil {
		log.Printf("Unable to read asset proxy with name %s: %s", name, result.Error.Error())
		return nil
	}
	return assetProxy
}
