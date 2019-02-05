package types

import (
	"bytes"
	"database/sql/driver"
	"fmt"
)

type AssetData []byte

var BitDexProxyID = [4]byte{93, 56, 142, 23}
var ERC20ProxyID = [4]byte{244, 114, 97, 176}
var ERC721ProxyID = [4]byte{2, 87, 23, 146}

func (data AssetData) ProxyId() [4]byte {
	result := [4]byte{}
	copy(result[:], data[0:4])
	return result
}

func (data AssetData) Address() *Address {
	address := &Address{}
	if data.IsType(BitDexProxyID) || data.IsType(ERC20ProxyID) || data.IsType(ERC721ProxyID) {
		copy(address[:], data[16:36])
	}
	return address
}

func (data AssetData) IsType(proxyId [4]byte) bool {
	return bytes.Equal(data[0:4], proxyId[:])
}

func (data AssetData) SupportedType() bool {
	return data.IsType(BitDexProxyID) || data.IsType(ERC20ProxyID) || data.IsType(ERC721ProxyID)
}

func (data AssetData) TokenID() *Uint256 {
	tokenID := &Uint256{}
	if data.IsType(ERC721ProxyID) {
		copy(tokenID[:], data[36:])
	}
	return tokenID
}

func (data AssetData) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%#x\"", data[:])), nil
}

func (data AssetData) Value() (driver.Value, error) {
	return []byte(data[:]), nil
}
