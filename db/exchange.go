package db

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/notegio/openrelay/types"
)

// Exchange contains information about supported exchange.
type Exchange struct {
	Address   *types.Address `gorm:"primary_key"`
	Network   uint64         `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ExchangeLookup contains helper maps for fast lookup.
type ExchangeLookup struct {
	db             *gorm.DB
	byAddressCache map[types.Address]uint64
	byNetworkCache map[uint64][]*types.Address
}

// GetExchangesByNetwork returns exchanges for specified network ID.
func (lookup *ExchangeLookup) GetExchangesByNetwork(network uint64) ([]*types.Address, error) {
	if addresses, ok := lookup.byNetworkCache[network]; ok {
		return addresses, nil
	}
	addresses := []*types.Address{}
	exchanges := []Exchange{}
	if err := lookup.db.Model(&Exchange{}).Where("network = ?", network).Find(&exchanges).Error; err != nil {
		return nil, err
	}
	for _, exchange := range exchanges {
		addresses = append(addresses, exchange.Address)
	}
	lookup.byNetworkCache[network] = addresses
	return addresses, nil
}

// GetNetworkByExchange returns network ID for specified exchange address.
func (lookup *ExchangeLookup) GetNetworkByExchange(address *types.Address) (uint64, error) {
	if network, ok := lookup.byAddressCache[*address]; ok {
		return network, nil
	}
	exchange := &Exchange{}
	if err := lookup.db.Model(&Exchange{}).Where("address = ?", address).First(exchange).Error; err != nil {
		return 0, err
	}
	lookup.byAddressCache[*address] = exchange.Network
	return exchange.Network, nil
}

// ExchangeIsKnown returns channel with single expecting value to be pushed.
func (lookup *ExchangeLookup) ExchangeIsKnown(address *types.Address) <-chan uint64 {
	result := make(chan uint64)
	go func(address *types.Address, result chan uint64) {
		networkID, _ := lookup.GetNetworkByExchange(address)
		result <- networkID
	}(address, result)
	return result
}

// NewExchangeLookup creates helper object improving exchanges reading.
func NewExchangeLookup(db *gorm.DB) *ExchangeLookup {
	return &ExchangeLookup{
		db,
		make(map[types.Address]uint64),
		make(map[uint64][]*types.Address),
	}
}
