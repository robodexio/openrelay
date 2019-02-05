package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/notegio/openrelay/affiliates"
	"github.com/notegio/openrelay/config"
	"github.com/notegio/openrelay/monitor/blocks"
	"github.com/notegio/openrelay/types"
	"gopkg.in/redis.v3"
)

func main() {
	redisURL := os.Args[1]
	lastRecordedBlockNumber := new(big.Int)
	lastRecordedBlockNumber.SetString(os.Args[2], 10)
	baseFeeString := os.Args[3]
	authorizedAddresses := os.Args[4:]
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	baseFeeService := config.NewBaseFee(redisClient)
	baseFeeInt := new(big.Int)
	baseFeeInt.SetString(baseFeeString, 10)
	baseFeeService.Set(baseFeeInt)

	affiliateService := affiliates.NewRedisAffiliateService(redisClient)
	for _, address := range authorizedAddresses {
		if addressBytes, err := hex.DecodeString(address[2:]); err == nil {
			addressArray := &types.Address{}
			copy(addressArray[:], addressBytes[:])
			affiliate := affiliates.NewAffiliate(baseFeeInt, 100)
			affiliateService.Set(addressArray, affiliate)
			fmt.Printf("Added address '%v'\n", hex.EncodeToString(addressArray[:]))
		}
	}

	if lastRecordedBlockNumber.Uint64() > 0 {
		blockRecorder := blocks.NewRedisBlockRecorder(redisClient, "newblocks::blocknumber")
		err := blockRecorder.Record(lastRecordedBlockNumber)
		if err != nil {
			fmt.Printf("Error: Unable to record block number %s to queue as last scanned block\n", lastRecordedBlockNumber)
		} else {
			fmt.Printf("Recorded block number %s to queue as last scanned block\n", lastRecordedBlockNumber)
		}
	}
}
