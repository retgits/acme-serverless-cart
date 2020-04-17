package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// ModifyCart modifies the entire cart
func ModifyCart(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	crt, err := acmeserverless.UnmarshalCart(string(ctx.Request.Body()))
	if err != nil {
		ErrorHandler(ctx, "ModifyCart", "UnmarshalCart", err)
		return
	}

	err = db.StoreItems(userID, crt.Items)
	if err != nil {
		ErrorHandler(ctx, "ModifyCart", "StoreItems", err)
		return
	}

	res := acmeserverless.UserIDResponse{
		UserID: userID,
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "ModifyCart", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
