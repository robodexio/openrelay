package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"

	accountsModule "github.com/notegio/openrelay/accounts"
	affiliatesModule "github.com/notegio/openrelay/affiliates"
	"github.com/notegio/openrelay/channels"
	poolModule "github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/types"
	"github.com/notegio/openrelay/zeroex"
)

// PostOrderConfig .
func PostOrderConfig(
	publisher channels.Publisher,
	accounts accountsModule.AccountService,
	affiliates affiliatesModule.AffiliateService,
	exchangeLookup ExchangeLookup,
	defaultFeeRecipient [20]byte,
) func(http.ResponseWriter, *http.Request, *poolModule.Pool) {

	return func(w http.ResponseWriter, r *http.Request, pool *poolModule.Pool) {

		// Check HTTP request method
		if r.Method != "POST" {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Unsupported HTTP request method",
			}, http.StatusBadRequest)
			return
		}

		// Check HTTP request content type
		var contentType string
		if contentTypeValue, ok := r.Header["Content-Type"]; ok {
			contentType = strings.Split(contentTypeValue[0], ";")[0]
		}
		if contentType != "application/json" {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Unsupported HTTP request content type",
			}, http.StatusBadRequest)
			return
		}

		// Parse HTTP request body content
		orderConfigRequest := &zeroex.OrderConfigRequest{}
		var contentBytes [4096]byte
		contentLength, err := r.Body.Read(contentBytes[:])
		if err != nil && err != io.EOF {
			log.Printf("Error reading content: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Error reading content",
			}, http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(contentBytes[:contentLength], &err); err != nil {
			log.Printf("Malformed JSON '%v': %v", string(contentBytes[:]), err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeMalformedJSON,
				Reason: "Malformed JSON",
			}, http.StatusBadRequest)
			return
		}

		// Check maker address
		makerAddressBytes, err := types.HexStringToBytes(orderConfigRequest.MakerAddress)
		if err != nil && orderConfigRequest.MakerAddress != "" {
			log.Printf("Malformed JSON '%v': %v", string(contentBytes[:]), err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "makerAddress",
					Code:   zeroex.ValidationErrorCodeIncorrectFormat,
					Reason: "Invalid address format",
				}},
			}, http.StatusBadRequest)
			return
		}
		makerAddress := &types.Address{}
		copy(makerAddress[:], makerAddressBytes[:])

		// Check exchange address
		exchangeAddressBytes, err := types.HexStringToBytes(orderConfigRequest.ExchangeAddress)
		if err != nil && orderConfigRequest.ExchangeAddress != "" {
			log.Printf("Malformed JSON '%v': %v", string(contentBytes[:]), err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "exchangeAddress",
					Code:   zeroex.ValidationErrorCodeIncorrectFormat,
					Reason: "Invalid address format",
				}},
			}, http.StatusBadRequest)
			return
		}
		exchangeAddress := &types.Address{}
		copy(exchangeAddress[:], exchangeAddressBytes[:])

		// Prepare fee recipient address
		feeRecipientAddress := &types.Address{}
		copy(feeRecipientAddress[:], defaultFeeRecipient[:])

		// Request network ID from exchange address asynchronously since this may have some latency
		chanNetworkID := exchangeLookup.ExchangeIsKnown(exchangeAddress)

		// Request the account from redis asynchronously since this may have some latency
		chanAccount := make(chan accountsModule.Account)
		chanAffiliate := make(chan affiliatesModule.Affiliate)
		go func() {
			chanAccount <- accounts.Get(makerAddress)
		}()
		go func() {
			feeRecipient, err := affiliates.Get(feeRecipientAddress)
			if err != nil {
				chanAffiliate <- nil
			} else {
				chanAffiliate <- feeRecipient
			}
		}()

		// Check network ID
		networkID := <-chanNetworkID
		if networkID == 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "exchangeContractAddress",
					Code:   zeroex.ValidationErrorCodeInvalidAddress,
					Reason: "Unknown exchangeContractAddress",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Check fee recipient address
		feeRecipient := <-chanAffiliate
		if feeRecipient == nil {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "feeRecipient",
					Code:   zeroex.ValidationErrorCodeInvalidAddress,
					Reason: "Invalid fee recpient",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Get pool fee
		poolFee, err := pool.Fee()
		if err != nil {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "pool",
					Code:   zeroex.ValidationErrorCodeInvalidAddress,
					Reason: "Pool error",
				}},
			}, http.StatusInternalServerError)
			return
		}

		// A fee recipient's Fee() value is the base fee for that recipient. A
		// maker's Discount() is the discount that recipient gets from the base
		// fee. Thus, the minimum fee required is pool.Fee() - maker.Discount()
		account := <-chanAccount
		minFee := new(big.Int)
		minFee.Sub(poolFee, account.Discount())

		// Prepare sender address to specify
		var senderAddressToSpecify string
		senderAddress, ok := pool.SenderAddresses[networkID]
		if ok {
			senderAddressToSpecify = fmt.Sprintf("%#x", senderAddress[:])
		} else {
			senderAddressToSpecify = fmt.Sprintf("%#x", emptyAddress[:])
		}

		// Prepare fee recipient address to specify
		feeRecipientAddressToSpecify := fmt.Sprintf("%#x", feeRecipientAddress[:])

		// Prepare order config response
		orderConfigResponse := &zeroex.OrderConfigResponse{
			SenderAddress:       senderAddressToSpecify,
			FeeRecipientAddress: feeRecipientAddressToSpecify,
			MakerFee:            minFee.Text(10),
			TakerFee:            "0",
		}
		orderConfigResponseBytes, err := json.Marshal(orderConfigResponse)
		if err != nil {
			log.Println("Unable to marshal order config response to JSON: %v", err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
			}, http.StatusInternalServerError)
		}

		// Everything is OK so respond with success HTTP status code and order config response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(orderConfigResponseBytes)
	}
}
