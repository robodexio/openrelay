package pool

import (
	"bytes"
	"fmt"
	"math/big"
	"net/http"
	urlModule "net/url"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/jinzhu/gorm"
	"github.com/notegio/openrelay/config"
	dbModule "github.com/notegio/openrelay/db"
	"github.com/notegio/openrelay/search"
	"github.com/notegio/openrelay/types"
	"gopkg.in/redis.v3"
)

var feeBaseUnits = big.NewInt(1000000000000000000)

type Pool struct {
	SearchTerms     string
	Expiration      uint64
	Nonce           uint
	FeeShare        string
	ID              []byte
	Limit           uint
	SenderAddresses types.NetworkAddressMap
	FilterAddresses types.NetworkAddressMap
	conn            bind.ContractCaller
	baseFee         config.BaseFee
}

func (pool *Pool) SetConn(conn bind.ContractCaller) {
	pool.conn = conn
}

func (pool *Pool) SetBaseFee(baseFee config.BaseFee) {
	pool.baseFee = baseFee
}

func (pool *Pool) CheckFilter(order *types.Order, networkid uint64) (bool, error) {
	if len(pool.FilterAddresses) == 0 {
		return true, nil
	}
	filterAddress, ok := pool.FilterAddresses[networkid]
	if !ok {
		// networkid is not supported by this pool, so neither is this order
		return false, nil
	}
	if bytes.Equal(filterAddress[:], make([]byte, 20)) {
		// If no filter contract is specified, everything is valid
		return true, nil
	}
	if pool.conn == nil {
		return false, fmt.Errorf("No connection set on pool")
	}
	return NewFilterContract(filterAddress, pool.conn).Filter(pool.ID, order)
}

func (pool Pool) Filter(query *gorm.DB) (*gorm.DB, error) {
	queryObject, err := urlModule.ParseQuery(pool.SearchTerms)
	if err != nil {
		return nil, err
	}
	query, errs := search.QueryFilter(query, queryObject)
	if errCount := len(errs); errCount > 0 {
		return nil, fmt.Errorf("Found %v errors in pool query string", errCount)
	}
	return query, nil
}

func (pool Pool) Count(db *gorm.DB) <-chan uint {
	result := make(chan uint)
	var value uint
	go func() {
		db.Model(&dbModule.Order{}).Where("pool_id = ?", pool.ID).Where("status = ?", dbModule.StatusOpen).Count(&value)
		result <- value
	}()
	return result
}

func (pool Pool) Fee() (*big.Int, error) {
	baseFee, err := pool.baseFee.Get()
	if err != nil {
		return nil, err
	}
	feeShare := new(big.Int)
	if _, ok := feeShare.SetString(pool.FeeShare, 10); !ok {
		// If the fee share is not a valid integer, just return the base fee
		return baseFee, nil
	}
	combined := new(big.Int).Mul(baseFee, feeShare)
	return new(big.Int).Div(combined, feeBaseUnits), nil
}

var poolRegex = regexp.MustCompile("^(/[^/]*)?/v2/")

func PoolDecorator(db *gorm.DB, fn func(http.ResponseWriter, *http.Request, types.Pool)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		match := poolRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 2 {
			poolName := strings.TrimPrefix(match[1], "/")
			pool := &Pool{}
			poolHash := sha3.NewKeccak256()
			poolHash.Write([]byte(poolName))
			if q := db.Model(&Pool{}).Where("ID = ?", poolHash.Sum(nil)).First(pool); q.Error != nil {
				if poolName != "" {
					w.WriteHeader(404)
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(fmt.Sprintf("{\"code\":102,\"reason\":\"Pool Not Found: %v\"}", q.Error.Error())))
					return
				}
				// If no pool was specified and no default pool is in the database,
				// just use an empty pool
			}
			fn(w, r, pool)
		} else {
			// Routing regex shouldn't get here
			w.WriteHeader(404)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fmt.Sprintf("{\"code\":102,\"reason\":\"Not Found\"}")))
			return
		}
	}
}

func PoolDecoratorBaseFee(db *gorm.DB, redisClient *redis.Client, fn func(http.ResponseWriter, *http.Request, *Pool)) func(http.ResponseWriter, *http.Request) {
	baseFee := config.NewBaseFee(redisClient)
	return func(w http.ResponseWriter, r *http.Request) {
		match := poolRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 2 {
			poolName := strings.TrimPrefix(match[1], "/")
			pool := &Pool{}
			poolHash := sha3.NewKeccak256()
			poolHash.Write([]byte(poolName))
			if q := db.Model(&Pool{}).Where("ID = ?", poolHash.Sum(nil)).First(pool); q.Error != nil {
				if poolName != "" {
					w.WriteHeader(404)
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(fmt.Sprintf("{\"code\":102,\"reason\":\"Pool Not Found: %v\"}", q.Error.Error())))
					return
				}
				// If no pool was specified and no default pool is in the database,
				// just use an empty pool
			}
			pool.baseFee = baseFee
			fmt.Printf("Pool: %v", pool)
			fn(w, r, pool)
		} else {
			// Routing regex shouldn't get here
			w.WriteHeader(404)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fmt.Sprintf("{\"code\":102,\"reason\":\"Not Found\"}")))
			return
		}
	}
}
