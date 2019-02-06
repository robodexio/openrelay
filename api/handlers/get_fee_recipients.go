package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/notegio/openrelay/affiliates"
	"github.com/notegio/openrelay/zeroex"
)

// FeeRecipientHandler .
func FeeRecipientHandler(affiliateService affiliates.AffiliateService) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryObject := r.URL.Query()
		page, perPage := extractPagination(&queryObject)
		affiliates, err := affiliateService.List()
		if err != nil {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: fmt.Sprintf("Internal server error: %s", err.Error()),
			}, http.StatusInternalServerError)
			return
		}
		total := len(affiliates)
		startIndex := int((page - 1) * perPage)
		if startIndex > total {
			startIndex = total
		}
		endIndex := int(page * perPage)
		if endIndex > total {
			endIndex = total
		}
		feeRecipientsPaginated := createPaginatedRecords(uint64(total), page, perPage, affiliates[startIndex:endIndex])
		response, err := json.Marshal(feeRecipientsPaginated)
		if err != nil {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: fmt.Sprintf("Internal server error: %s", err.Error()),
			}, http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
