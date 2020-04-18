package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// ModifyCartItem modifies a single item in a cart
func ModifyCartItem(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("userid").(string)

	cartItems, err := db.GetItems(userID)
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "GetItems", err)
		return
	}

	item, err := acmeserverless.UnmarshalItem(ctx.Request.Body())
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "UnmarshalItem", err)
		return
	}

	for idx, cci := range cartItems {
		if *cci.ItemID == *item.ItemID {
			cartItems[idx] = item
		}
	}

	err = db.StoreItems(userID, cartItems)
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "StoreItems", err)
		return
	}

	res := acmeserverless.UserIDResponse{
		UserID: userID,
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "ModifyCartItem", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
