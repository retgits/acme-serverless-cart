package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// AddItemToCart adds an item to the cart of a user
func AddItemToCart(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	// Unmarshal the item
	item, err := acmeserverless.UnmarshalItem(ctx.Request.Body())
	if err != nil {
		ErrorHandler(ctx, "AddItemToCart", "UnmarshalItem", err)
		return
	}

	// Add the item
	err = db.AddItem(userID, item)
	if err != nil {
		ErrorHandler(ctx, "AddItemToCart", "AddItem", err)
		return
	}

	// Create a response
	res := acmeserverless.UserIDResponse{
		UserID: userID,
	}

	// Create the byte payload for the response
	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "AddItemToCart", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
