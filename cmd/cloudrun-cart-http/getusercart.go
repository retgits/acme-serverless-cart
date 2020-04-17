package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetUserCart gets the cart of a single user
func GetUserCart(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	items, err := db.GetItems(userID)
	if err != nil {
		ErrorHandler(ctx, "GetUserCart", "GetItems", err)
		return
	}

	ct := acmeserverless.Cart{
		Items:  items,
		UserID: userID,
	}

	payload, err := ct.Marshal()
	if err != nil {
		ErrorHandler(ctx, "GetUserCart", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
