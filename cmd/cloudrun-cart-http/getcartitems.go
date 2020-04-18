package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetCartItems gets the contents of a single cart
func GetCartItems(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	items, err := db.GetItems(userID)
	if err != nil {
		ErrorHandler(ctx, "GetCartItems", "GetItem", err)
		return
	}

	ct := acmeserverless.Cart{
		Items:  items,
		UserID: userID,
	}

	payload, err := ct.Marshal()
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
