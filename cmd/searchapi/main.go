package main

import (
	"github.com/notegio/openrelay/search"
	"github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/channels"
	"github.com/notegio/openrelay/blockhash"
	"github.com/notegio/openrelay/affiliates"
	dbModule "github.com/notegio/openrelay/db"
	"net/http"
	"gopkg.in/redis.v3"
	"os"
	"log"
	// "github.com/rs/cors"
	"strconv"
	"regexp"
)

func corsDecorator(fn func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		if r.Method == "OPTIONS" {
			if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
				w.Header().Set("Access-Control-Allow-Headers", h)
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.WriteHeader(200)
			}
		} else {
			fn(w, r)
		}
	}
}

type route struct {
    pattern *regexp.Regexp
    handler http.Handler
}

type regexpHandler struct {
    routes []*route
}

func (h *regexpHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
    h.routes = append(h.routes, &route{pattern, handler})
}

func (h *regexpHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
    h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handler)})
}

func (h *regexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    for _, route := range h.routes {
        if route.pattern.MatchString(r.URL.Path) {
            route.handler.ServeHTTP(w, r)
            return
        }
    }
    // no pattern matched; send 404 response
    http.NotFound(w, r)
}

func main() {
	redisURL := os.Args[1]
	blockChannel := os.Args[2]
	db, err := dbModule.GetDB(os.Args[3], os.Args[4])
	if err != nil {
		log.Fatalf("Could not open database connection: %v", err.Error())
	}
	port := "8082"
	for _, arg := range os.Args[5:] {
		if _, err := strconv.Atoi(arg); err == nil {
			// If the argument is castable as an integer,
			port = arg
		}
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	blockChannelConsumer, err := channels.ConsumerFromURI(blockChannel, redisClient)
	if err != nil {
		log.Fatalf("Error establishing block channel: %v", err.Error())
	}
	blockHash := blockhash.NewChanneledBlockHash(blockChannelConsumer)
	searchHandler := corsDecorator(search.BlockHashDecorator(blockHash, pool.PoolDecorator(db, search.SearchHandler(db))))
	orderHandler := corsDecorator(search.BlockHashDecorator(blockHash, search.OrderHandler(db)))
	orderBookHandler := corsDecorator(search.BlockHashDecorator(blockHash, pool.PoolDecorator(db, search.OrderBookHandler(db))))
	feeRecipientsHandler := corsDecorator(search.BlockHashDecorator(blockHash, search.FeeRecipientHandler(affiliates.NewRedisAffiliateService(redisClient))))
	pairHandler := corsDecorator(search.PairHandler(db))

	mux := &regexpHandler{[]*route{}}
	mux.HandleFunc(regexp.MustCompile("^(/[^/]+)?/v2/orders$"), searchHandler)
	mux.HandleFunc(regexp.MustCompile("^(/[^/]+)?/v2/order/$"), orderHandler)
	mux.HandleFunc(regexp.MustCompile("^(/[^/]+)?/v2/asset_pairs$"), pairHandler)
	mux.HandleFunc(regexp.MustCompile("^(/[^/]+)?/v2/orderbook$"), orderBookHandler)
	mux.HandleFunc(regexp.MustCompile("^(/[^/]+)?/v2/fee_recipients$"), feeRecipientsHandler)
	mux.HandleFunc(regexp.MustCompile("^/_hc$"), search.HealthCheckHandler(db, blockHash))
	log.Printf("Order Search Serving on :%v", port)
	http.ListenAndServe(":"+port, mux)
}
