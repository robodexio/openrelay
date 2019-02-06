package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/fatih/color"
	"gopkg.in/redis.v3"

	"github.com/notegio/openrelay/accounts"
	"github.com/notegio/openrelay/affiliates"
	"github.com/notegio/openrelay/api/handlers"
	"github.com/notegio/openrelay/blockhash"
	"github.com/notegio/openrelay/channels"
	dbModule "github.com/notegio/openrelay/db"
	"github.com/notegio/openrelay/search"
	"github.com/rs/cors"
)

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type router struct {
	routes []*route
}

func (router *router) Handler(pattern *regexp.Regexp, handler http.Handler) {
	router.routes = append(router.routes, &route{pattern, handler})
}

func (router *router) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	router.routes = append(router.routes, &route{pattern, http.HandlerFunc(handler)})
}

func (router *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range router.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	// no pattern matched; send 404 response
	http.NotFound(w, r)
}

func main() {

	// Read service parameters from command arguments
	dbConnectionURL := os.Args[1]
	dbConnectionPassword := os.Args[2]
	redisConnectionURL := os.Args[3]
	redisBlockQueueURL := os.Args[4]
	redisOutputQueueURL := os.Args[5]
	serviceFeeRecipientString := os.Args[6]
	var servicePort string
	if len(os.Args) > 7 {
		servicePort = os.Args[7]
	} else {
		servicePort = "8080"
	}

	// Prepare fee recipient address as 20 bytes slice
	serviceFeeRecipientBytes, err := hex.DecodeString(serviceFeeRecipientString[2:])
	handleError("Unable to parse fee recipient address", err)
	serviceFeeRecipientAddress := [20]byte{}
	copy(serviceFeeRecipientAddress[:], serviceFeeRecipientBytes[:])

	// Initialize connection to PostgreSQL DB
	db, err := dbModule.GetDB(dbConnectionURL, dbConnectionPassword)
	handleError("Unable to connect to PostgreSQL DB", err)

	// Initialize connection to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisConnectionURL,
	})
	// TODO: Check connection is initialized correctly

	// Create listener of the Redis block queue
	listenerBlock, err := channels.ConsumerFromURI(redisBlockQueueURL, redisClient)
	handleError("Unable to create listener of the Redis block queue", err)
	blockHash := blockhash.NewChanneledBlockHash(listenerBlock)

	// Create publisher to the Redis output queue
	publisher, err := channels.PublisherFromURI(redisOutputQueueURL, redisClient)
	handleError("Unable to create publisher to the Redis output queue", err)

	// Create helper service objects
	affiliateService := affiliates.NewRedisAffiliateService(redisClient)
	accountService := accounts.NewRedisAccountService(redisClient)
	exchangeLookup := dbModule.NewExchangeLookup(db)

	// Prepare handlers
	handlerGetAssetPairs := handlers.GetAssetPairs(db)
	handlerGetOrders := handlers.PoolDecorator(db, search.SearchHandler(db))
	handlerGetOrder := handlers.OrderHandler(db)
	handlerGetOrderBook := handlers.PoolDecorator(db, search.OrderBookHandler(db))
	handlerGetFeeRecipients := handlers.FeeRecipientHandler(affiliateService)
	handlerPostOrderConfig := handlers.PoolDecoratorBaseFee(db, redisClient, handlers.PostOrderConfig(
		publisher,
		accountService,
		affiliateService,
		exchangeLookup,
		serviceFeeRecipientAddress,
	))
	handlerPostOrder := handlers.PoolDecoratorBaseFee(db, redisClient, handlers.PostOrder(
		publisher,
		accountService,
		affiliateService,
		exchangeLookup,
	))
	handlerGetHealthCheck := handlers.GetHealthCheck(db, redisClient, blockHash)

	// Prepare HTTP handler which handles all incoming HTTP requests
	handlerCases := []struct {
		m string
		p string
		f func(http.ResponseWriter, *http.Request)
	}{
		{m: "GET", p: "^(/[^/]+)?/0x/v2/asset_pairs$", f: handlerGetAssetPairs},
		{m: "GET", p: "^(/[^/]+)?/0x/v2/orders$", f: handlerGetOrders}, // paginated, makerAssetProxyId, takerAssetProxyId, makerAssetAddress, takerAssetAddress, ...
		{m: "GET", p: "^(/[^/]+)?/0x/v2/order/$", f: handlerGetOrder},
		{m: "GET", p: "^(/[^/]+)?/0x/v2/orderbook$", f: handlerGetOrderBook}, // paginated, baseAssetData, quoteAssetData
		{m: "GET", p: "^(/[^/]+)?/0x/v2/fee_recipients$", f: handlerGetFeeRecipients},
		{m: "POST", p: "^(/[^/]+)?/0x/v2/order_config$", f: handlerPostOrderConfig},
		{m: "POST", p: "^(/[^/]+)?/0x/v2/order$", f: handlerPostOrder},
		{m: "GET", p: "^/_hc$", f: handlerGetHealthCheck},
	}
	handlerMuxer := &router{[]*route{}}
	for _, c := range handlerCases {
		handlerMuxer.HandleFunc(regexp.MustCompile(c.p), c.f)
	}
	httpHandler := cors.Default().Handler(handlerMuxer)

	// Start listening incoming HTTP requests
	log.Printf("Service API started listening port %s", servicePort)
	http.ListenAndServe(fmt.Sprintf(":%s", servicePort), httpHandler)
}

var colorError = color.New(color.FgHiRed)

func handleError(reason string, err error) {
	if err != nil {
		message := fmt.Sprintf("[ERROR] %s: %s\n", reason, err.Error())
		colorError.Print(message)
		panic(message)
	}
}
