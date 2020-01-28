package datastore

import cart "github.com/retgits/acme-serverless-cart"

// Manager ...
type Manager interface {
	GetItems(userID string) (cart.Items, error)
	AddItem(userID string, i cart.Item) error
	AllCarts() (cart.Carts, error)
	ClearCart(userID string) error
	StoreItems(userID string, i cart.Items) error
	ItemsInCart(userID string) (int64, error)
	ValueInCart(userID string) (float64, error)
}
