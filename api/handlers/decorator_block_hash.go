package handlers

import (
	"net/http"
	"strings"

	"github.com/notegio/openrelay/blockhash"
)

// BlockHashDecorator .
func BlockHashDecorator(
	blockHash blockhash.BlockHash,
	fn func(http.ResponseWriter, *http.Request),
) func(http.ResponseWriter, *http.Request) {
	// Start the go routines, if necessary
	blockHash.Get()
	return func(w http.ResponseWriter, r *http.Request) {
		queryObject := r.URL.Query()
		hash := queryObject.Get("blockhash")
		if hash == "" {
			queryObject.Set("blockhash", strings.Trim(blockHash.Get(), "\""))
			url := *r.URL
			url.RawQuery = queryObject.Encode()
			w.Header().Set("Cache-Control", "max-age=5, public")
			http.Redirect(w, r, (&url).RequestURI(), http.StatusTemporaryRedirect)
			return
		}
		fn(w, r)
	}
}
