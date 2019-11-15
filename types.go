package cart

import "encoding/json"

func UnmarshalCart(data string) (Cart, error) {
	var r Cart
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

func (r *Cart) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Carts) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Carts []Cart

type Cart struct {
	Items  []Item `json:"cart"`
	Userid string `json:"userid"`
}

type Item struct {
	Description string  `json:"description"`
	Itemid      string  `json:"itemid"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Quantity    int64   `json:"quantity"`
}

func UnmarshalItem(data []byte) (Item, error) {
	var r Item
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CartTotal) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type CartTotal struct {
	CartItemTotal int64  `json:"cartitemtotal"`
	UserID        string `json:"userid"`
}

func (r *CartValue) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type CartValue struct {
	CartTotal float64 `json:"carttotal"`
	UserID    string  `json:"userid"`
}
