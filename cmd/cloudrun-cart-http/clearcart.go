package main

import (
	"net/http"

	"github.com/valyala/fasthttp"
)

// ClearCart removes all items from a cart
func ClearCart(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	// Remove the cart
	err := db.ClearCart(userID)
	if err != nil {
		ErrorHandler(ctx, "ClearCart", "ClearCart", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
}
