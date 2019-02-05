package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	dbModule "github.com/notegio/openrelay/db"
	"github.com/notegio/openrelay/types"
	"github.com/notegio/openrelay/zeroex"
)

const (
	paginationPageDefault    = "1"
	paginationPerPageDefault = "20"
)

var emptyAddress = types.Address{}

// ExchangeLookup .
type ExchangeLookup interface {
	ExchangeIsKnown(*types.Address) <-chan uint
}

func respondError(w http.ResponseWriter, e *zeroex.Error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errBytes, err := json.Marshal(e)
	if err != nil {
		log.Printf(err.Error())
	}
	w.Write(errBytes)
}

func extractPagination(query *url.Values) (uint64, uint64, error) {
	queryPage := query.Get("page")
	if len(queryPage) == 0 {
		queryPage = paginationPageDefault
	}
	queryPerPage := query.Get("perPage")
	if len(queryPerPage) == 0 {
		queryPerPage = paginationPerPageDefault
	}
	page, err := strconv.Atoi(queryPage)
	if err != nil {
		return 0, 0, err
	}
	if page <= 0 {
		return 0, 0, fmt.Errorf("Query param 'page' should be greater than 0")
	}
	perPage, err := strconv.Atoi(queryPerPage)
	if err != nil {
		return 0, 0, err
	}
	if perPage <= 0 {
		return 0, 0, fmt.Errorf("Query param 'perPage' should be greater than 0")
	}
	return uint64(page), uint64(perPage), nil
}

func createPaginatedRecords(
	total uint64,
	page uint64,
	perPage uint64,
	records interface{},
) *zeroex.PaginatedResponse {
	return &zeroex.PaginatedResponse{total, page, perPage, records}
}

func createPaginatedAssetPairs(
	total uint64,
	page uint64,
	perPage uint64,
	records []dbModule.Pair,
) *zeroex.PadinatedAssetPairs {
	result := &zeroex.PadinatedAssetPairs{total, page, perPage, zeroex.AssetPairs{}}
	for _, record := range records {
		result.Records = append(result.Records, &zeroex.AssetPair{
			AssetDataA: zeroex.AssetData{
				MinAmount: "1",
				MaxAmount: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
				Precision: 3,
				AssetData: fmt.Sprintf("%#x", []byte(record.TokenA)),
			},
			AssetDataB: zeroex.AssetData{
				MinAmount: "1",
				MaxAmount: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
				Precision: 3,
				AssetData: fmt.Sprintf("%#x", []byte(record.TokenB)),
			},
		})
	}
	return result
}

func createPaginatedOrders(
	total uint64,
	page uint64,
	perPage uint64,
	records zeroex.OrderExs,
) *zeroex.PadinatedOrders {
	return &zeroex.PadinatedOrders{total, page, perPage, records}
}
