// Package cart contains all events that the Cart service
// in the ACME Serverless Fitness Shop can send and receive.
package cart

import "encoding/json"

const (
	// Domain is the domain where the services reside
	Domain = "Cart"
)

// Carts is a slice of Cart objects
type Carts []Cart

// Marshal returns the JSON encoding of Carts
func (r *Carts) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Cart represents a shoppingcart for a user
// of the ACME Serverless Fitness Shop
type Cart struct {
	// Items is a slice of Item objects, each being a single
	// object in the cart of the user
	Items []Item `json:"cart"`

	// UserID is the unique identifier of the user in the
	// ACME Serverless Fitness Shop
	UserID string `json:"userid"`
}

// Marshal returns the JSON encoding of Cart
func (r *Cart) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// UnmarshalCart parses the JSON-encoded data and stores the result in
// a Cart
func UnmarshalCart(data string) (Cart, error) {
	var r Cart
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

// Items is a slice of Item objects
type Items []Item

// Marshal returns the JSON encoding of Items
func (r *Items) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// UnmarshalItems parses the JSON-encoded data and stores the result in
// an Items object
func UnmarshalItems(data string) (Items, error) {
	var r Items
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

// Item represents the items that the ACME Serverless Fitness Shop
// user has in their shopping cart
type Item struct {
	// Description is a description of the items
	Description string `json:"description"`

	// ItemID is the unique identifier of the item
	ItemID string `json:"itemid"`

	// Name is the name of the item
	Name string `json:"name"`

	// Price is the monetairy value of the item
	Price float64 `json:"price"`

	// Quantity is how many of the item the user has in their cart
	Quantity int64 `json:"quantity"`
}

// UnmarshalItem parses the JSON-encoded data and stores the result
// in an Item
func UnmarshalItem(data []byte) (Item, error) {
	var r Item
	err := json.Unmarshal(data, &r)
	return r, err
}

// CartTotal represents how many items the user currently has in their cart
type CartTotal struct {
	// CartItemTotal is the number of items
	CartItemTotal int64 `json:"cartitemtotal"`

	// UserID is the unique identifier of the user in the
	// ACME Serverless Fitness Shop
	UserID string `json:"userid"`
}

// Marshal returns the JSON encoding of CartTotal
func (r *CartTotal) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// CartValue represents the total value of all items currently in the
// cart of the iser
type CartValue struct {
	// CartTotal is the value of items
	CartTotal float64 `json:"carttotal"`

	// UserID is the unique identifier of the user in the
	// ACME Serverless Fitness Shop
	UserID string `json:"userid"`
}

// Marshal returns the JSON encoding of CartValue
func (r *CartValue) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// UserIDResponse returns the UserID
type UserIDResponse struct {
	// UserID is the unique identifier of the user in the
	// ACME Serverless Fitness Shop
	UserID string `json:"userid"`
}

// Marshal returns the JSON encoding on UserIDResponse
func (r *UserIDResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}
