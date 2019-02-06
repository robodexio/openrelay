package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	accountsModule "github.com/notegio/openrelay/accounts"
	affiliatesModule "github.com/notegio/openrelay/affiliates"
	"github.com/notegio/openrelay/channels"
	poolModule "github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/types"
	"github.com/notegio/openrelay/zeroex"
)

// PostOrder .
func PostOrder(
	publisher channels.Publisher,
	accounts accountsModule.AccountService,
	affiliates affiliatesModule.AffiliateService,
	exchangeLookup ExchangeLookup,
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
		order := types.Order{}
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
		if err := json.Unmarshal(contentBytes[:contentLength], &order); err != nil {
			log.Printf("Malformed JSON '%v': %v", string(contentBytes[:]), err.Error())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeMalformedJSON,
				Reason: "Malformed JSON",
			}, http.StatusBadRequest)
			return
		}

		// Request network ID from exchange address asynchronously since this may have some latency
		chanNetworkID := exchangeLookup.ExchangeIsKnown(order.ExchangeAddress)

		// Check order assets
		if !order.MakerAssetData.SupportedType() {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "makerAssetData",
					Code:   zeroex.ValidationErrorCodeUnsupportedOption,
					Reason: fmt.Sprintf("Unsupported asset type: %#x", order.MakerAssetData.ProxyId()),
				}},
			}, http.StatusBadRequest)
			return
		}
		if !order.TakerAssetData.SupportedType() {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "takerAssetData",
					Code:   zeroex.ValidationErrorCodeUnsupportedOption,
					Reason: fmt.Sprintf("Unsupported asset type: %#x", order.TakerAssetData.ProxyId()),
				}},
			}, http.StatusBadRequest)
			return
		}

		// Check order signature type
		if !order.Signature.Supported() {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "signature",
					Code:   zeroex.ValidationErrorCodeInvalidSignatureOrHash,
					Reason: "Unsupported signature type",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Verify order signature
		if !order.Signature.Verify(order.Maker, order.Hash()) {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "signature",
					Code:   zeroex.ValidationErrorCodeInvalidSignatureOrHash,
					Reason: "Signature validation failed",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Check order expiration time
		timeNow := big.NewInt(time.Now().Unix())
		if timeNow.Cmp(order.ExpirationTimestampInSec.Big()) > 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "expirationUnixTimestampSec",
					Code:   zeroex.ValidationErrorCodeValueOutOfRange,
					Reason: "Order already expired",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Check order expiration time
		timeFuture := big.NewInt(0).Add(timeNow, big.NewInt(31536000000))
		if timeFuture.Cmp(order.ExpirationTimestampInSec.Big()) < 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "expirationUnixTimestampSec",
					Code:   zeroex.ValidationErrorCodeValueOutOfRange,
					Reason: "Expiration in distant future",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Check order asset amounts
		if big.NewInt(0).Cmp(order.TakerAssetAmount.Big()) == 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "TakerAssetAmount",
					Code:   zeroex.ValidationErrorCodeValueOutOfRange,
					Reason: "takerAssetAmount must be > 0",
				}},
			}, http.StatusBadRequest)
			return
		}
		if big.NewInt(0).Cmp(order.MakerAssetAmount.Big()) == 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
					Field:  "MakerAssetAmount",
					Code:   zeroex.ValidationErrorCodeValueOutOfRange,
					Reason: "makerAssetAmount must be > 0",
				}},
			}, http.StatusBadRequest)
			return
		}

		// Request the account from redis asynchronously since this may have some latency
		chanAccount := make(chan accountsModule.Account)
		chanAffiliate := make(chan affiliatesModule.Affiliate)
		go func() {
			chanAccount <- accounts.Get(order.Maker)
		}()
		go func() {
			feeRecipient, err := affiliates.Get(order.FeeRecipient)
			if err != nil {
				log.Printf("Error retrieving fee recipient: %v", err.Error())
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

		// Check sender address
		if len(pool.SenderAddresses) != 0 {
			exchangeAddress := pool.SenderAddresses[networkID][:]
			exchangeAddressEmpty := bytes.Equal(exchangeAddress, emptyAddress[:])
			exchangeAddressValid := bytes.Equal(exchangeAddress, order.SenderAddress[:])
			if !exchangeAddressEmpty && !exchangeAddressValid {
				respondError(w, &zeroex.Error{
					Code:   zeroex.ErrorCodeValidationFailed,
					Reason: "Validation Failed",
					ValidationErrors: []zeroex.ValidationError{zeroex.ValidationError{
						Field:  "senderAddress",
						Code:   zeroex.ValidationErrorCodeInvalidAddress,
						Reason: "Invalid sender for this order pool / network",
					}},
				}, http.StatusBadRequest)
				return
			}
		}

		// Check pool expiration
		if pool.Expiration > 0 && pool.Expiration < timeNow.Uint64() {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeOrderSubmissionDisabled,
				Reason: "Order Pool Expired",
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
					Reason: "Invalid fee recipient",
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
					Reason: err.Error(),
				}},
			}, http.StatusInternalServerError)
			return
		}

		// A pool's Fee() value is the base fee for that pool. A maker's Discount()
		// is the discount that recipient gets from the base fee. Thus, the minimum
		// fee required is pool.Fee() - maker.Discount()
		account := <-chanAccount
		minFee := new(big.Int)
		makerFee := new(big.Int).SetBytes(order.MakerFee[:])
		takerFee := new(big.Int).SetBytes(order.TakerFee[:])
		totalFee := new(big.Int).Add(makerFee, takerFee)
		minFee.Sub(poolFee, account.Discount())
		if totalFee.Cmp(minFee) < 0 {
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
				ValidationErrors: []zeroex.ValidationError{
					zeroex.ValidationError{
						Field:  "makerFee",
						Code:   zeroex.ValidationErrorCodeValueOutOfRange,
						Reason: "Total fee must be at least: " + minFee.Text(10),
					},
					zeroex.ValidationError{
						Field:  "takerFee",
						Code:   zeroex.ValidationErrorCodeValueOutOfRange,
						Reason: "Total fee must be at least: " + minFee.Text(10),
					},
				},
			}, http.StatusBadRequest)
			return
		}

		// Check if account is denied
		if account.Blacklisted() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, "")
			return
		}

		// Send validated order to redis
		order.PoolID = pool.ID
		orderBytes := order.Bytes()
		if ok := publisher.Publish(string(orderBytes[:])); !ok {
			log.Println("Unable to publish order with hash %#x", order.Hash())
			respondError(w, &zeroex.Error{
				Code:   zeroex.ErrorCodeValidationFailed,
				Reason: "Validation Failed",
			}, http.StatusInternalServerError)
		}

		// Everything is OK so just respond with success HTTP status code
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "")
	}
}
