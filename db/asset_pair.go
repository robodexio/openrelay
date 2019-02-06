package db

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/notegio/openrelay/types"
)

// AssetPair contains information about supported asset pair.
type AssetPair struct {
	ID           uint64 `gorm:"primary_key;AUTO_INCREMENT"`
	AssetSymbolA string `gorm:"primary_key"`
	AssetSymbolB string `gorm:"primary_key"`
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	AssetA       *Asset `gorm:"foreignkey:asset_symbol_a"`
	AssetB       *Asset `gorm:"foreignkey:asset_symbol_b"`
}

// Save records the asset pair in the database.
func (assetPair *AssetPair) Save(db *gorm.DB) *gorm.DB {
	log.Printf("Attempting to save asset pair with symbols %s-%s...", assetPair.AssetSymbolA, assetPair.AssetSymbolB)
	result := db.Model(&AssetPair{}).FirstOrCreate(assetPair)
	if result.Error != nil {
		log.Printf("Unable to save asset pair with symbols %s-%s: %s", assetPair.AssetSymbolA, assetPair.AssetSymbolB, result.Error.Error())
	}
	return result
}

// GetAssetPairs returns an unfiltered list of asset pairs, limited by a limit and offset.
func GetAssetPairs(db *gorm.DB, offset uint64, limit uint64, networkID uint64) ([]*AssetPair, uint64, error) {
	assetPairs := []*AssetPair{}
	var total uint64
	if err := db.Model(&AssetPair{}).Count(&total).Error; err != nil {
		return assetPairs, total, err
	}
	if total == 0 {
		return assetPairs, total, nil
	}
	if err := db.Preload("AssetA").Preload("AssetB").Offset(offset).Limit(limit).Find(&assetPairs).Error; err != nil {
		return assetPairs, total, err
	}
	return assetPairs, total, nil
}

// GetAssetPairsByAssetData returns a list of asset pairs, filtered to include only asset pairs
// that include specified asset data and limited by a limit and offset.
func GetAssetPairsByAssetData(db *gorm.DB, assetData types.AssetData, offset uint64, limit uint64, networkID uint64) ([]*AssetPair, uint64, error) {
	assetDataBytes := []byte(assetData[:])
	assetPairs := []*AssetPair{}
	var total uint64
	query := db.Model(&AssetPair{}).
		Joins("JOIN assets AS asset_a ON asset_a.symbol = asset_pairs.asset_symbol_a").
		Joins("JOIN assets AS asset_b ON asset_b.symbol = asset_pairs.asset_symbol_b").
		Joins("JOIN asset_proxies AS asset_proxy_a ON asset_proxy_a.id = asset_a.proxy_id").
		Joins("JOIN asset_proxies AS asset_proxy_b ON asset_proxy_b.id = asset_b.proxy_id").
		Joins("JOIN exchanges AS exchange_a ON exchange_a.address = asset_proxy_a.exchange_address").
		Joins("JOIN exchanges AS exchange_b ON exchange_b.address = asset_proxy_b.exchange_address").
		Where("exchange_a.network = ? AND exchange_b.network = ?", networkID, networkID).
		Where("asset_a.data = ? OR asset_b.data = ?", assetDataBytes, assetDataBytes)
	if err := query.Count(&total).Error; err != nil {
		return assetPairs, total, err
	}
	if total == 0 {
		return assetPairs, total, nil
	}
	if err := query.Offset(offset).Limit(limit).Preload("AssetA").Preload("AssetB").Find(&assetPairs).Error; err != nil {
		return assetPairs, total, err
	}
	return assetPairs, total, nil
}

// GetAssetPairsByAssetDatas returns a list of asset pairs, filtered to include only asset pairs
// that include both specified asset datas. There should only be one distinct combination of both
// asset pairs, so there is no offset or limit, but it still returns a list to provide the same
// return value as the other retrieval methods.
func GetAssetPairsByAssetDatas(db *gorm.DB, assetDataA types.AssetData, assetDataB types.AssetData, networkID uint64) ([]*AssetPair, uint64, error) {
	assetDataBytesA := []byte(assetDataA[:])
	assetDataBytesB := []byte(assetDataB[:])
	assetPairs := []*AssetPair{}
	var total uint64
	query := db.Model(&AssetPair{}).
		Joins("JOIN assets AS asset_a ON asset_a.symbol = asset_pairs.asset_symbol_a").
		Joins("JOIN assets AS asset_b ON asset_b.symbol = asset_pairs.asset_symbol_b").
		Joins("JOIN asset_proxies AS asset_proxy_a ON asset_proxy_a.id = asset_a.proxy_id").
		Joins("JOIN asset_proxies AS asset_proxy_b ON asset_proxy_b.id = asset_b.proxy_id").
		Joins("JOIN exchanges AS exchange_a ON exchange_a.address = asset_proxy_a.exchange_address").
		Joins("JOIN exchanges AS exchange_b ON exchange_b.address = asset_proxy_b.exchange_address").
		Where("exchange_a.network = ? AND exchange_b.network = ?", networkID, networkID).
		Where("(asset_a.data = ? AND asset_b.data = ?) OR (asset_a.data = ? AND asset_b.data = ?)",
			assetDataBytesA, assetDataBytesB, assetDataBytesB, assetDataBytesA,
		)
	if err := query.Count(&total).Error; err != nil {
		return assetPairs, total, err
	}
	if total == 0 {
		return assetPairs, total, nil
	}
	if err := query.Preload("AssetA").Preload("AssetB").Find(&assetPairs).Error; err != nil {
		return assetPairs, total, err
	}
	return assetPairs, total, nil
}

// GetAssetPairByID reads the asset pair by assets symbols from the database.
func GetAssetPairByID(db *gorm.DB, id uint64) (*AssetPair, error) {
	assetPair := &AssetPair{}
	if result := db.
		Where("id = ?", id).
		Preload("AssetA").Preload("AssetB").
		First(assetPair); result.Error != nil {
		log.Printf("Unable to read asset pair with ID %d: %s", id, result.Error.Error())
		return nil, result.Error
	}
	return assetPair, nil
}

// GetAssetPairByAssetSymbols reads the asset pair by assets symbols from the database.
func GetAssetPairByAssetSymbols(db *gorm.DB, symbolA string, symbolB string) (*AssetPair, error) {
	assetPair := &AssetPair{}
	if result := db.
		Where("asset_symbol_a = ? AND asset_symbol_b = ?", symbolA, symbolB).
		Preload("AssetA").Preload("AssetB").
		First(assetPair); result.Error != nil {
		log.Printf("Unable to read asset pair with symbols %s-%s: %s", symbolA, symbolB, result.Error.Error())
		return nil, result.Error
	}
	return assetPair, nil
}

// GetAssetPairByAssetDatas reads the asset pair by assets datas from the database.
func GetAssetPairByAssetDatas(db *gorm.DB, assetDataA types.AssetData, assetDataB types.AssetData) (*AssetPair, error) {
	assetPair := &AssetPair{}
	if result := db.Model(&AssetPair{}).
		Joins("JOIN assets AS asset_a ON asset_a.symbol = asset_pairs.asset_symbol_a").
		Joins("JOIN assets AS asset_b ON asset_b.symbol = asset_pairs.asset_symbol_b").
		Where("asset_a.data = ? AND asset_b.data = ?", []byte(assetDataA[:]), []byte(assetDataB[:])).
		First(assetPair); result.Error != nil {
		log.Printf("Unable to read asset pair with specified asset datas: %s", result.Error.Error())
		return nil, result.Error
	}
	return assetPair, nil
}
