package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/jinzhu/gorm"
	"github.com/notegio/openrelay/config"
	pl "github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/types"
	redis "gopkg.in/redis.v3"
)

var poolRegex = regexp.MustCompile("^(/[^/]*)?/v2/")

// PoolDecorator .
func PoolDecorator(
	db *gorm.DB,
	fn func(http.ResponseWriter, *http.Request, types.Pool),
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		match := poolRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 2 {
			poolName := strings.TrimPrefix(match[1], "/")
			pool := &pl.Pool{}
			poolHash := sha3.NewKeccak256()
			poolHash.Write([]byte(poolName))
			if q := db.Model(&pl.Pool{}).Where("ID = ?", poolHash.Sum(nil)).First(pool); q.Error != nil {
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

// PoolDecoratorBaseFee .
func PoolDecoratorBaseFee(
	db *gorm.DB,
	redisClient *redis.Client,
	fn func(http.ResponseWriter, *http.Request, *pl.Pool),
) func(http.ResponseWriter, *http.Request) {
	baseFee := config.NewBaseFee(redisClient)
	return func(w http.ResponseWriter, r *http.Request) {

		match := poolRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 2 {
			poolName := strings.TrimPrefix(match[1], "/")
			pool := &pl.Pool{}
			poolHash := sha3.NewKeccak256()
			poolHash.Write([]byte(poolName))
			if q := db.Model(&pl.Pool{}).Where("ID = ?", poolHash.Sum(nil)).First(pool); q.Error != nil {
				if poolName != "" {
					w.WriteHeader(404)
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(fmt.Sprintf("{\"code\":102,\"reason\":\"Pool Not Found: %v\"}", q.Error.Error())))
					return
				}
				// If no pool was specified and no default pool is in the database,
				// just use an empty pool
			}
			pool.SetBaseFee(baseFee)
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
