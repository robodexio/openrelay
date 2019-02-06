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
		queryObject := r.URL.Query()
		queryAssetDataA := queryObject.Get("assetDataA")
		queryAssetDataB := queryObject.Get("assetDataB")
		queryNetworkID := queryObject.Get("networkId")

		// Read pagination query params
		page, perPage := extractPagination(&queryObject)
		offset := (page - 1) * perPage

		// Some adjustments
		if len(queryAssetDataA) == 0 && len(queryAssetDataB) > 0 {
			queryAssetDataA = queryAssetDataB
			queryAssetDataB = ""
		}
		networkID, err := strconv.ParseUint(queryNetworkID, 10, 64)
		if err != nil {
			networkID = 1
		}

		// Request asset pairs from DB
		var assetPairs []*dbModule.AssetPair
		var total uint64
		if len(queryAssetDataA) == 0 {
			assetPairs, total, err = dbModule.GetAssetPairs(db, offset, perPage, networkID)
			if err != nil {
				log.Printf("Unable to get asset pairs from DB: %v", err.Error())
				respondError(w, &zeroex.Error{
					Code:   zeroex.ErrorCodeValidationFailed,
					Reason: "Unable to get asset pairs from DB",
				}, http.StatusBadRequest)
				return
			}
		} else {
			assetDataA, err := common.HexToAssetData(queryAssetDataA)
			if err != nil {
				log.Printf("Unable to parse asset data specified in query: %v", err.Error())
				respondError(w, &zeroex.Error{
					Code:   zeroex.ErrorCodeValidationFailed,
					Reason: "Unable to parse asset data specified in query",
				}, http.StatusBadRequest)
				return
			}
			if len(queryAssetDataB) == 0 {
				assetPairs, total, err = dbModule.GetAssetPairsByAssetData(db, assetDataA, offset, perPage, networkID)
			} else {
				assetDataB, err := common.HexToAssetData(queryAssetDataB)
				if err != nil {
					log.Printf("Unable to parse asset data specified in query: %v", err.Error())
					respondError(w, &zeroex.Error{
						Code:   zeroex.ErrorCodeValidationFailed,
						Reason: "Unable to parse asset data specified in query",
					}, http.StatusBadRequest)
					return
				}
				assetPairs, total, err = dbModule.GetAssetPairsByAssetDatas(db, assetDataA, assetDataB, networkID)
			}
			if err != nil {
				log.Printf("Unable to get asset pairs from DB: %v", err.Error())
				respondError(w, &zeroex.Error{
					Code:   zeroex.ErrorCodeValidationFailed,
					Reason: "Unable to get asset pairs from DB",
				}, http.StatusBadRequest)
				return
			}
		}

		// Prepare response
		paginatedAssetPairs := createPaginatedAssetPairs(total, page, perPage, assetPairs)
		response, err := json.Marshal(paginatedAssetPairs)
		if err != nil {
			log.Printf("Internal error: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
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
