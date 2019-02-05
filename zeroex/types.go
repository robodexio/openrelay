package zeroex

// HTTPStatus is HTTP status code which can be returned from 0x API requests.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#errors
type HTTPStatus uint16

const (
	// HTTPStatusOK returned in case of success request.
	HTTPStatusOK HTTPStatus = 200
	// HTTPStatusCreated returned in case of success request which creates something.
	HTTPStatusCreated HTTPStatus = 201
	// HTTPStatusAccepted returned in case of request which is correct but cannot be performed.
	HTTPStatusAccepted HTTPStatus = 202
	// HTTPStatusBadRequest returned in case of invalid request format.
	HTTPStatusBadRequest HTTPStatus = 400
	// HTTPStatusNotFound returned in case when requested resource is not found.
	HTTPStatusNotFound HTTPStatus = 404
	// HTTPStatusTooManyRequests returned in case when rate limit exceeded.
	HTTPStatusTooManyRequests HTTPStatus = 429
	// HTTPStatusInternalError returned in case when internal server error happened.
	HTTPStatusInternalError HTTPStatus = 500
	// HTTPStatusNotImplemented returned in case of not implemented request.
	HTTPStatusNotImplemented HTTPStatus = 501
)

// ErrorCode is error code which can be returned from 0x API requests in case of some error.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#error-reporting-format
type ErrorCode uint16

const (
	// ErrorCodeValidationFailed .
	ErrorCodeValidationFailed ErrorCode = 100
	// ErrorCodeMalformedJSON .
	ErrorCodeMalformedJSON ErrorCode = 101
	// ErrorCodeOrderSubmissionDisabled .
	ErrorCodeOrderSubmissionDisabled ErrorCode = 102
	// ErrorCodeThrottled .
	ErrorCodeThrottled ErrorCode = 103
)

// ValidationErrorCode is validation error code which can be returned from 0x API requests in case of validation error.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#error-reporting-format
type ValidationErrorCode uint16

const (
	// ValidationErrorCodeRequiredField .
	ValidationErrorCodeRequiredField ValidationErrorCode = 1000
	// ValidationErrorCodeIncorrectFormat .
	ValidationErrorCodeIncorrectFormat ValidationErrorCode = 1001
	// ValidationErrorCodeInvalidAddress .
	ValidationErrorCodeInvalidAddress ValidationErrorCode = 1002
	// ValidationErrorCodeAddressNotSupported .
	ValidationErrorCodeAddressNotSupported ValidationErrorCode = 1003
	// ValidationErrorCodeValueOutOfRange .
	ValidationErrorCodeValueOutOfRange ValidationErrorCode = 1004
	// ValidationErrorCodeInvalidSignatureOrHash .
	ValidationErrorCodeInvalidSignatureOrHash ValidationErrorCode = 1005
	// ValidationErrorCodeUnsupportedOption .
	ValidationErrorCodeUnsupportedOption ValidationErrorCode = 1006
)

// Error contains information about error handled during API request.
type Error struct {
	Code             ErrorCode         `json:"code"`
	Reason           string            `json:"reason"`
	ValidationErrors []ValidationError `json:"validationErrors,omitempty"`
}

// ValidationError contains information about order validation error.
type ValidationError struct {
	Field  string              `json:"field"`
	Code   ValidationErrorCode `json:"code"`
	Reason string              `json:"reason"`
}

// AssetData contains information specific to an asset.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#get-v2asset_pairs
type AssetData struct {
	MinAmount string `json:"minAmount"`
	MaxAmount string `json:"maxAmount"`
	Precision uint8  `json:"precision"`
	AssetData string `json:"assetData"`
}

// AssetPair contains pair of an assets.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#get-v2asset_pairs
type AssetPair struct {
	AssetDataA AssetData `json:"assetDataA"`
	AssetDataB AssetData `json:"assetDataB"`
}

// Order contains ZeroEx signed order.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#post-v2order
type Order struct {
	MakerAddress          string `json:"makerAddress"`
	TakerAddress          string `json:"takerAddress"`
	SenderAddress         string `json:"senderAddress"`
	FeeRecipientAddress   string `json:"feeRecipientAddress"`
	MakerFee              string `json:"makerFee"`
	TakerFee              string `json:"takerFee"`
	MakerAssetAmount      string `json:"makerAssetAmount"`
	TakerAssetAmount      string `json:"takerAssetAmount"`
	MakerAssetData        string `json:"makerAssetData"`
	TakerAssetData        string `json:"takerAssetData"`
	Salt                  string `json:"salt"`
	ExchangeAddress       string `json:"exchangeAddress"`
	ExpirationTimeSeconds string `json:"expirationTimeSeconds"`
	Signature             string `json:"signature"`
}

// OrderEx contains ZeroEx signed order with a meta data.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#get-v2orderorderhash
type OrderEx struct {
	Order    *Order        `json:"order"`
	Metadata OrderMetadata `json:"metaData"`
}

// OrderMetadata contains some additional to ZeroEx order information.
type OrderMetadata struct {
	Hash                      string  `json:"hash"`
	FeeRate                   float64 `json:"feeRate"`
	Status                    int64   `json:"status"`
	TakerAssetAmountRemaining string  `json:"takerAssetAmountRemaining"`
}

// OrderBook contains the orderbook for a given asset pair.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#get-v2orderbook
type OrderBook struct {
	BaseTokenAddress  string          `json:"baseTokenAddress"`
	QuoteTokenAddress string          `json:"quoteTokenAddress"`
	Bids              PadinatedOrders `json:"bids"`
	Asks              PadinatedOrders `json:"asks"`
}

// PaginatedResponse contains base fields of padinated request.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#pagination
type PaginatedResponse struct {
	Total   uint64      `json:"total"`
	Page    uint64      `json:"page"`
	PerPage uint64      `json:"perPage"`
	Records interface{} `json:"records"`
}

// PadinatedAssetPairs contains padinated array of asset pairs.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#get-v2asset_pairs
type PadinatedAssetPairs struct {
	Total   uint64     `json:"total"`
	Page    uint64     `json:"page"`
	PerPage uint64     `json:"perPage"`
	Records AssetPairs `json:"records"`
}

// PadinatedOrders contains padinated array of orders.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#get-v2orderbook
type PadinatedOrders struct {
	Total   uint64   `json:"total"`
	Page    uint64   `json:"page"`
	PerPage uint64   `json:"perPage"`
	Records OrderExs `json:"records"`
}

// OrderConfigRequest contains ZeroEx order config request.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#post-v2order_config
type OrderConfigRequest struct {
	MakerAddress          string `json:"makerAddress"`
	TakerAddress          string `json:"takerAddress"`
	MakerAssetAmount      string `json:"makerAssetAmount"`
	TakerAssetAmount      string `json:"takerAssetAmount"`
	MakerAssetData        string `json:"makerAssetData"`
	TakerAssetData        string `json:"takerAssetData"`
	ExchangeAddress       string `json:"exchangeAddress"`
	ExpirationTimeSeconds string `json:"expirationTimeSeconds"`
}

// OrderConfigResponse contains ZeroEx order config response.
// https://github.com/0xProject/standard-relayer-api/blob/master/http/v2.md#post-v2order_config
type OrderConfigResponse struct {
	SenderAddress       string `json:"senderAddress"`
	FeeRecipientAddress string `json:"feeRecipientAddress"`
	MakerFee            string `json:"makerFee"`
	TakerFee            string `json:"takerFee"`
}

// AssetPairs contains array of AssetPair objects.
type AssetPairs []*AssetPair

// Orders contains array of Order objects.
type Orders []*Order

// OrderExs contains array of OrderEx objects.
type OrderExs []*OrderEx
