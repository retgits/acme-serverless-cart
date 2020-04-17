package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetCartValue gets the total monetary value in the cart of a user
func GetCartValue(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	value, err := db.ValueInCart(userID)
	if err != nil {
		ErrorHandler(ctx, "GetCartValue", "ValueInCart", err)
		return
	}

	ct := acmeserverless.CartValueTotal{
		CartTotal: value,
		UserID:    userID,
	}

	payload, err := ct.Marshal()
	if err != nil {
		ErrorHandler(ctx, "GetCartValue", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
