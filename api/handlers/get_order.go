package handlers

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"

	dbModule "github.com/notegio/openrelay/db"
	"github.com/notegio/openrelay/zeroex"
)

var orderRegex = regexp.MustCompile(".*/order/0[xX]([0-9a-fA-F]+)")

// OrderHandler .
func OrderHandler(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		match := orderRegex.FindStringSubmatch(r.URL.Path)
		if len(match) == 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Malformed order hash",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "orderHash",
					Code:   zeroex.ValidationErrorCodeIncorrectFormat,
					Reason: "Order hash is not specified or specified incorrectly",
				}},
			}, http.StatusNotFound)
			return
		}
		orderHashHex := match[1]
		orderHash, err := hex.DecodeString(orderHashHex)
		if err != nil {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Malformed order hash",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "orderHash",
					Code:   zeroex.ValidationErrorCodeIncorrectFormat,
					Reason: "Order hash is specified incorrectly",
				}},
			}, http.StatusBadRequest)
			return
		}
		order := &dbModule.Order{}
		query := db.Model(&dbModule.Order{}).Where("order_hash = ?", orderHash).First(order)
		if query.Error != nil {
			if query.Error.Error() == "record not found" {
				respondError(w, &zeroex.Error{
					Code:   zeroex.ErrorCodeValidationFailed,
					Reason: fmt.Sprintf("Order with specified hash %#x is not found", orderHash),
				}, http.StatusNotFound)
			} else {
				respondError(w, &zeroex.Error{
					Code:   zeroex.ErrorCodeValidationFailed,
					Reason: fmt.Sprintf("Internal server error happened during seaching requested order: %s", query.Error.Error()),
				}, http.StatusInternalServerError)
			}
			return
		}
		var acceptHeader string
		if acceptValue, ok := r.Header["Accept"]; ok {
			acceptHeader = strings.Split(acceptValue[0], ";")[0]
		} else {
			acceptHeader = "unknown"
		}
		response, contentType, err := formatSingleResponse(order, acceptHeader)
		if err != nil {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: fmt.Sprintf("Internal server error happened during preparing order response: %s", err.Error()),
			}, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
