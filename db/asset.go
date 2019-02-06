package db

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/notegio/openrelay/types"
)

// Asset contains information about supported asset.
type Asset struct {
	Symbol         string           `gorm:"primary_key"`
	Name           string           `gorm:"index"`
	Address        *types.Address   `gorm:"index"`
	Decimals       uint16           `gorm:"index"`
	ProxyID        []byte           `gorm:"index"`
	Data           *types.AssetData `gorm:"index"`
	Precision      uint16
	MinTradeAmount *types.Uint256
	MaxTradeAmount *types.Uint256
	ZeroEx         bool
	Active         bool
	Quote          bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Proxy          *AssetProxy `gorm:"foreignkey:proxy_id"`
}

// Save records the asset in the database.
func (asset *Asset) Save(db *gorm.DB) *gorm.DB {
	log.Printf("Attempting to save asset with symbol %s...", asset.Symbol)
	result := db.Model(&Asset{}).FirstOrCreate(asset)
	if result.Error != nil {
		log.Printf("Unable to save asset with symbol %s: %s", asset.Symbol, result.Error.Error())
	}
	return result
}

// GetAssetBySymbol reads the asset by it's symbol from the database.
func GetAssetBySymbol(db *gorm.DB, symbol string) *Asset {
	asset := &Asset{}
	if result := db.Where("symbol = ?", symbol).First(asset); result.Error != nil {
		log.Printf("Unable to read asset with symbol %s: %s", symbol, result.Error.Error())
		return nil
	}
	return asset
}

// GetAssetByName reads the asset by it's name from the database.
func GetAssetByName(db *gorm.DB, name string) *Asset {
	asset := &Asset{}
	if result := db.Where("name = ?", name).First(asset); result.Error != nil {
		log.Printf("Unable to read asset with name %s: %s", name, result.Error.Error())
		return nil
	}
	return asset
}
