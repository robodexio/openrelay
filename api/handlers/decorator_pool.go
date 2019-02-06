package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/jinzhu/gorm"
	redis "gopkg.in/redis.v3"

	"github.com/notegio/openrelay/config"
	poolModule "github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/types"
	"github.com/notegio/openrelay/zeroex"
)

var poolRegex = regexp.MustCompile("^(/[^/]*)?/0x/v2/")

// PoolDecorator .
func PoolDecorator(
	db *gorm.DB,
	fn func(http.ResponseWriter, *http.Request, types.Pool),
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		match := poolRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 2 {
			pool := &poolModule.Pool{}
			poolName := strings.TrimPrefix(match[1], "/")
			poolHash := sha3.NewKeccak256()
			poolHash.Write([]byte(poolName))
			poolHashBytes := poolHash.Sum(nil)
			if result := db.Model(&poolModule.Pool{}).Where("id = ?", poolHashBytes).First(pool); result.Error != nil {
				if len(poolName) > 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(fmt.Sprintf(
						"{\"code\":%d,\"reason\":\"Pool Not Found: %v\"}",
						zeroex.ErrorCodeOrderSubmissionDisabled,
						result.Error.Error(),
					)))
					return
				}
				// If no pool was specified and no default pool is in the database, just use an empty pool
			}
			fn(w, r, pool)
		} else {
			// Routing regex shouldn't get here
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf(
				"{\"code\":%d,\"reason\":\"Not Found\"}",
				zeroex.ErrorCodeOrderSubmissionDisabled,
			)))
			return
		}
	}
}

// PoolDecoratorBaseFee .
func PoolDecoratorBaseFee(
	db *gorm.DB,
	redisClient *redis.Client,
	fn func(http.ResponseWriter, *http.Request, *poolModule.Pool),
) func(http.ResponseWriter, *http.Request) {
	baseFee := config.NewBaseFee(redisClient)
	return func(w http.ResponseWriter, r *http.Request) {
		match := poolRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 2 {
			pool := &poolModule.Pool{}
			poolName := strings.TrimPrefix(match[1], "/")
			poolHash := sha3.NewKeccak256()
			poolHash.Write([]byte(poolName))
			poolHashBytes := poolHash.Sum(nil)
			if result := db.Model(&poolModule.Pool{}).Where("id = ?", poolHashBytes).First(pool); result.Error != nil {
				if len(poolName) > 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(fmt.Sprintf(
						"{\"code\":%d,\"reason\":\"Pool Not Found: %v\"}",
						zeroex.ErrorCodeOrderSubmissionDisabled,
						result.Error.Error(),
					)))
					return
				}
				// If no pool was specified and no default pool is in the database, just use an empty pool
			}
			pool.SetBaseFee(baseFee)
			fmt.Printf("Pool: %v", pool)
			fn(w, r, pool)
		} else {
			// Routing regex shouldn't get here
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf(
				"{\"code\":%d,\"reason\":\"Not Found\"}",
				zeroex.ErrorCodeOrderSubmissionDisabled,
			)))
			return
		}
	}
}
