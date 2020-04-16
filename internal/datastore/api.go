// Package datastore contains the interfaces that the Cart service
// in the ACME Serverless Fitness Shop needs to store and retrieve data.
// In order to add a new service, the Manager interface
// needs to be implemented.
package datastore

import acmeserverless "github.com/retgits/acme-serverless"

// Manager is the interface that describes the methods the
// data store needs to implement to be able to work with
// the ACME Serverless Fitness Shop.
type Manager interface {
	GetItems(userID string) (acmeserverless.CartItems, error)
	AddItem(userID string, i acmeserverless.CartItem) error
	AllCarts() (acmeserverless.Carts, error)
	ClearCart(userID string) error
	StoreItems(userID string, i acmeserverless.CartItems) error
	ItemsInCart(userID string) (int64, error)
	ValueInCart(userID string) (float64, error)
}
