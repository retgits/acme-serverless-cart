package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetTotalItems gets the total number of items in a cart
func GetTotalItems(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	items, err := db.ItemsInCart(userID)
	if err != nil {
		ErrorHandler(ctx, "GetTotalItems", "ItemsInCart", err)
		return
	}

	ct := acmeserverless.CartItemTotal{
		CartItemTotal: items,
		UserID:        userID,
	}

	payload, err := ct.Marshal()
	if err != nil {
		ErrorHandler(ctx, "GetTotalItems", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
