package main

import (
	"net/http"

	"github.com/valyala/fasthttp"
)

// GetAllCarts gets all carts that are available in the database
func GetAllCarts(ctx *fasthttp.RequestCtx) {
	// Get all carts
	carts, err := db.AllCarts()
	if err != nil {
		ErrorHandler(ctx, "GetAllCarts", "AllCarts", err)
		return
	}

	// Create the byte payload for the response
	payload, err := carts.Marshal()
	if err != nil {
		ErrorHandler(ctx, "GetAllCarts", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
