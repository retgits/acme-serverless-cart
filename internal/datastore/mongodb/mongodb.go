// Package mongodb leverages cross-platform document-oriented database program. Classified as a
// NoSQL database program, MongoDB uses JSON-like documents with schema.
package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/retgits/acme-serverless-cart/internal/datastore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The pointer to MongoDB provides the API operation methods for making requests to MongoDB.
// This specifically creates a single instance of the MongoDB service which can be reused if the
// container stays warm.
var dbs *mongo.Collection

// manager is an empty struct that implements the methods of the
// Manager interface.
type manager struct{}

// init creates the connection to MongoDB.
func init() {
	username := os.Getenv("MONGO_USERNAME")
	password := os.Getenv("MONGO_PASSWORD")
	hostname := os.Getenv("MONGO_HOSTNAME")
	port := os.Getenv("MONGO_PORT")

	connString := fmt.Sprintf("mongodb+srv://%s:%s@%s:%s", username, password, hostname, port)
	if strings.HasSuffix(connString, ":") {
		connString = connString[:len(connString)-1]
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connString))
	if err != nil {
		log.Fatalf("error connecting to MongoDB: %s", err.Error())
	}
	dbs = client.Database("acmeserverless").Collection("cart")
}

// New creates a new datastore manager using Amazon DynamoDB as backend
func New() datastore.Manager {
	return manager{}
}

// GetItems retrieves all items for a single user from DynamoDB based on the userID
func (m manager) GetItems(userID string) (acmeserverless.CartItems, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	res := dbs.FindOne(ctx, bson.D{{"SK", userID}})

	raw, err := res.DecodeBytes()
	if err != nil {
		return nil, fmt.Errorf("unable to decode bytes: %s", err.Error())
	}

	payload := raw.Lookup("Payload").StringValue()

	if len(payload) < 5 {
		return make(acmeserverless.CartItems, 0), nil
	}

	return acmeserverless.UnmarshalItems(raw.Lookup("Payload").StringValue())
}

// AddItem adds a new item for the user to the cart
func (m manager) AddItem(userID string, i acmeserverless.CartItem) error {
	items, err := m.GetItems(userID)
	if err != nil {
		return err
	}

	items = append(items, i)
	cc, err := items.Marshal()
	if err != nil {
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = dbs.UpdateOne(ctx, bson.D{{"SK", userID}}, bson.D{{"$set", bson.D{{"Payload", string(cc)}}}})

	return err
}

// AllCarts retrieves all carts from DynamoDB
func (m manager) AllCarts() (acmeserverless.Carts, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cursor, err := dbs.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	var results []bson.M

	if err = cursor.All(ctx, &results); err != nil {
		log.Fatal(err)
	}

	carts := make(acmeserverless.Carts, 0)

	for _, result := range results {
		cartContent, err := acmeserverless.UnmarshalItems(result["Payload"].(string))
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshalling cart data: %s", err.Error()))
			continue
		}

		carts = append(carts, acmeserverless.Cart{
			Items:  cartContent,
			UserID: result["SK"].(string),
		})
	}

	return carts, nil
}

// ClearCart sets the cart for a user to an empty JSON string
func (m manager) ClearCart(userID string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err := dbs.UpdateOne(ctx, bson.D{{"SK", userID}}, bson.D{{"$set", bson.D{{"Payload", ""}}}})

	return err
}

// StoreItems saves the cart items from a single user into Amazon DynamoDB
func (m manager) StoreItems(userID string, i acmeserverless.CartItems) error {
	payload, err := i.Marshal()
	if err != nil {
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = dbs.UpdateOne(ctx, bson.D{{"SK", userID}}, bson.D{{"$set", bson.D{{"Payload", string(payload)}}}})

	return err
}

// ItemsInCart gets the number of items in a cart for the user
func (m manager) ItemsInCart(userID string) (int64, error) {
	items, err := m.GetItems(userID)
	if err != nil {
		return 0, err
	}

	numItems := int64(0)

	for _, ci := range items {
		numItems = numItems + ci.Quantity
	}

	return numItems, nil
}

// ValueInCart gets the value of the items in a cart for the user
func (m manager) ValueInCart(userID string) (float64, error) {
	items, err := m.GetItems(userID)
	if err != nil {
		return 0, err
	}

	value := float64(0)

	for _, ci := range items {
		value = value + (float64(ci.Quantity) * ci.Price)
	}

	return value, nil
}
