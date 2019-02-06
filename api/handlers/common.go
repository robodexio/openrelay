package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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
	ExchangeIsKnown(*types.Address) <-chan uint64
}

func getFormattedOrder(dbOrder *dbModule.Order) *zeroex.OrderEx {
	jsonOrder, err := json.Marshal(dbOrder.Order)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}
	order := zeroex.Order{}
	err = json.Unmarshal(jsonOrder, &order)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}
	return &zeroex.OrderEx{
		Order: &order,
		Metadata: zeroex.OrderMetadata{
			fmt.Sprintf("%#x", dbOrder.OrderHash[:]),
			dbOrder.FeeRate,
			dbOrder.Status,
			new(big.Int).Sub(dbOrder.TakerAssetAmount.Big(), dbOrder.TakerAssetAmountFilled.Big()).String(),
		},
	}
}

func formatResponse(dbOrders []*dbModule.Order, format string, total uint64, page uint64, perPage uint64) ([]byte, string, error) {
	if format == "application/octet-stream" {
		response := []byte{}
		for _, dbOrder := range dbOrders {
			response = append(response, dbOrder.Bytes()[:]...)
		}
		return response, "application/octet-stream", nil
	}
	orders := []*zeroex.OrderEx{}
	for _, dbOrder := range dbOrders {
		orders = append(orders, getFormattedOrder(dbOrder))
	}
	ordersPaginated := createPaginatedRecords(total, page, perPage, orders)
	response, err := json.Marshal(ordersPaginated)
	return response, "application/json", err
}

func formatSingleResponse(dbOrder *dbModule.Order, format string) ([]byte, string, error) {
	if format == "application/octet-stream" {
		return dbOrder.Bytes()[:], "application/octet-stream", nil
	}
	order := getFormattedOrder(dbOrder)
	response, err := json.Marshal(order)
	return response, "application/json", err
}

func extractPagination(queryObject *url.Values) (uint64, uint64) {
	queryPage := queryObject.Get("page")
	if len(queryPage) == 0 {
		queryPage = paginationPageDefault
	}
	queryPerPage := queryObject.Get("perPage")
	if len(queryPerPage) == 0 {
		queryPerPage = paginationPerPageDefault
	}
	page, err := strconv.ParseUint(queryPage, 10, 64)
	if err != nil || page <= 0 {
		log.Printf("Unable to extract query parameter 'page': %v", err.Error())
		page = 1
	}
	perPage, err := strconv.ParseUint(queryPerPage, 10, 64)
	if err != nil || perPage <= 0 {
		log.Printf("Unable to extract query parameter 'page': %v", err.Error())
		perPage = 1
	}
	return page, perPage
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
	records []*dbModule.AssetPair,
) *zeroex.PadinatedAssetPairs {
	result := &zeroex.PadinatedAssetPairs{total, page, perPage, zeroex.AssetPairs{}}
	for _, record := range records {
		result.Records = append(result.Records, &zeroex.AssetPair{
			AssetDataA: zeroex.AssetData{
				MinAmount: record.AssetA.MinTradeAmount.String(),
				MaxAmount: record.AssetA.MaxTradeAmount.String(),
				Precision: record.AssetA.Precision,
				AssetData: fmt.Sprintf("%#x", *record.AssetA.Data),
			},
			AssetDataB: zeroex.AssetData{
				MinAmount: record.AssetB.MinTradeAmount.String(),
				MaxAmount: record.AssetB.MaxTradeAmount.String(),
				Precision: record.AssetB.Precision,
				AssetData: fmt.Sprintf("%#x", *record.AssetB.Data),
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

func respondError(w http.ResponseWriter, e *zeroex.Error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errBytes, err := json.Marshal(e)
	if err != nil {
		log.Printf(err.Error())
	}
	w.Write(errBytes)
}
