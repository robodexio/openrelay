package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jinzhu/gorm"

	"github.com/notegio/openrelay/common"
	dbModule "github.com/notegio/openrelay/db"
	"github.com/notegio/openrelay/zeroex"
)

// GetAssetPairs .
func GetAssetPairs(db *gorm.DB) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		// Try to find all possible query params
		query := r.URL.Query()
		queryAssetDataA := query.Get("assetDataA")
		queryAssetDataB := query.Get("assetDataB")
		queryNetworkID := query.Get("networkId")

		// Read pagination query params
		page, perPage, err := extractPagination(&query)
		pageOffset := (page - 1) * perPage
		if err != nil {
			log.Printf("Unable to extract pagination query params: %v", err.Error())
			respondError(w, &zeroex.Error{
				zeroex.ErrorCodeValidationFailed,
				"Unable to extract pagination query params",
				nil,
			}, http.StatusBadRequest)
			return
		}

		// Some adjustments
		if queryAssetDataA == "" && queryAssetDataB != "" {
			queryAssetDataA = queryAssetDataB
			queryAssetDataB = ""
		}
		networkID, err := strconv.Atoi(queryNetworkID)
		if err != nil {
			networkID = 1
		}

		// Request asset pairs from DB
		var assetPairs []dbModule.Pair
		var count int
		if queryAssetDataA == "" {
			assetPairs, count, err = dbModule.GetAllTokenPairs(db, int(pageOffset), int(perPage), networkID)
			if err != nil {
				log.Printf("Unable to get asset pairs from DB: %v", err.Error())
				respondError(w, &zeroex.Error{
					zeroex.ErrorCodeValidationFailed,
					"Unable to get asset pairs from DB",
					nil,
				}, http.StatusBadRequest)
				return
			}
		} else {
			assetDataA, err := common.HexToAssetData(queryAssetDataA)
			if err != nil {
				log.Printf("Unable to parse asset data specified in query: %v", err.Error())
				respondError(w, &zeroex.Error{
					zeroex.ErrorCodeValidationFailed,
					"Unable to parse asset data specified in query",
					nil,
				}, http.StatusBadRequest)
				return
			}
			if queryAssetDataB == "" {
				assetPairs, count, err = dbModule.GetTokenAPairs(db, assetDataA, int(pageOffset), int(perPage), networkID)
			} else {
				assetDataB, err := common.HexToAssetData(queryAssetDataB)
				if err != nil {
					log.Printf("Unable to parse asset data specified in query: %v", err.Error())
					respondError(w, &zeroex.Error{
						zeroex.ErrorCodeValidationFailed,
						"Unable to parse asset data specified in query",
						nil,
					}, http.StatusBadRequest)
					return
				}
				assetPairs, count, err = dbModule.GetTokenABPairs(db, assetDataA, assetDataB, networkID)
			}
			if err != nil {
				log.Printf("Unable to parse asset data specified in query: %v", err.Error())
				respondError(w, &zeroex.Error{
					zeroex.ErrorCodeValidationFailed,
					"Unable to parse asset data specified in query",
					nil,
				}, http.StatusBadRequest)
				return
			}
		}

		// Prepare response
		paginatedAssetPairs := createPaginatedAssetPairs(uint64(count), page, perPage, assetPairs)
		response, err := json.Marshal(paginatedAssetPairs)
		if err != nil {
			log.Printf("Internal error: %v", err.Error())
			respondError(w, &zeroex.Error{
				zeroex.ErrorCodeValidationFailed,
				"Internal error",
				nil,
			}, http.StatusInternalServerError)
			return
		}

		// Everything is OK so respond with success HTTP status code and response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
