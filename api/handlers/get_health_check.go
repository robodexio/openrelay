package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/notegio/openrelay/blockhash"
	"github.com/notegio/openrelay/zeroex"
	redis "gopkg.in/redis.v3"
)

type healthCheck struct {
	Time      []string
	BlockHash string
}

// GetHealthCheck .
func GetHealthCheck(
	db *gorm.DB,
	redisClient *redis.Client,
	blockHash blockhash.BlockHash,
) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		// Check connection to Redis
		t, err := redisClient.Time().Result()
		if err != nil {
			log.Printf("Internal error: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeOrderSubmissionDisabled,
				Reason: "Internal error",
			}, http.StatusInternalServerError)
			return
		}

		// Check connection to PostgreSQL DB
		if err := db.Raw("SELECT 1").Error; err != nil {
			log.Printf("Internal error: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeOrderSubmissionDisabled,
				Reason: "Internal error",
			}, http.StatusInternalServerError)
			return
		}

		// Check last recorded block hash
		hash := strings.Trim(blockHash.Get(), "\"")
		if len(hash) == 0 {
			log.Printf("Internal error: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeOrderSubmissionDisabled,
				Reason: "Internal error",
			}, http.StatusInternalServerError)
			return
		}

		// Prepare response
		response, err := json.Marshal(&healthCheck{
			Time:      t,
			BlockHash: hash,
		})
		if err != nil {
			log.Printf("Internal error: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeOrderSubmissionDisabled,
				Reason: "Internal error",
			}, http.StatusInternalServerError)
			return
		}

		// Everything is OK so respond with success HTTP status code and response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
