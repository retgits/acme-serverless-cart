package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetCartItems gets the number of items in a cart
func GetCartItems(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	items, err := db.ItemsInCart(userID)
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "ItemsInCart", err)
		return
	}

	ct := acmeserverless.CartItemTotal{
		CartItemTotal: items,
		UserID:        userID,
	}

	payload, err := ct.Marshal()
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
