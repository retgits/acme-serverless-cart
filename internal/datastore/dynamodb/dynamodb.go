package dynamodb

import (
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	cart "github.com/retgits/acme-serverless-cart"
	"github.com/retgits/acme-serverless-cart/internal/datastore"
)

// Create a single instance of the dynamoDB service
// which can be reused if the container stays warm
var dbs *dynamodb.DynamoDB

type manager struct{}

// init creates the connection to dynamoDB. If the environment variable
// DYNAMO_URL is set, the connection is made to that URL instead of
// relying on the AWS SDK to provide the URL
func init() {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	if len(os.Getenv("DYNAMO_URL")) > 0 {
		awsSession.Config.Endpoint = aws.String(os.Getenv("DYNAMO_URL"))
	}

	dbs = dynamodb.New(awsSession)
}

// New creates a new datastore manager using Amazon DynamoDB as backend
func New() datastore.Manager {
	return manager{}
}

// GetItems retrieves all items for a single user from DynamoDB based on the userID
func (m manager) GetItems(userID string) (cart.Items, error) {
	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = CART SK = ID
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("CART"),
	}
	km[":id"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	// Create the QueryInput
	qi := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		KeyConditionExpression:    aws.String("PK = :type AND SK = :id"),
		ExpressionAttributeValues: km,
	}

	// Execute the DynamoDB query
	qo, err := dbs.Query(qi)
	if err != nil {
		return user.User{}, err
	}

	// Return an error if no data was found
	if so.Items[0]["Payload"].S == nil {
		return nil, fmt.Errorf("no items found with for user with id %s", userID)
	}

	str := *so.Items[0]["Payload"].S
	return cart.UnmarshalItems(str)
}

// AddItem adds a new item for the user to the cart
func (m manager) AddItem(userID string, i cart.Item) error {
	items, err := m.GetItems(userID)
	if err != nil {
		return err
	}

	items = append(items, i)
	cc, err := items.Marshal()
	if err != nil {
		return err
	}

	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("CART"),
	}
	km[":id"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":payload"] = &dynamodb.AttributeValue{
		S: aws.String(string(cc)),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET Payload = :payload"),
	}

	_, err = dbs.UpdateItem(uii)
	return err
}

// AllCarts retrieves all carts from DynamoDB
func (m manager) AllCarts() (cart.Carts, error) {
	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = CART
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("CART"),
	}

	// Create the QueryInput
	qi := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		KeyConditionExpression:    aws.String("PK = :type"),
		ExpressionAttributeValues: km,
	}

	qo, err := dbs.Query(qi)
	if err != nil {
		return nil, err
	}

	// Return an error if no data was found
	if len(qo.Items) == 0 {
		return nil, fmt.Errorf("no item data found")
	}

	carts := make(cart.Carts, 0)

	for _, ct := range so.Items {
		str := *ct["Payload"].S

		cartContent, err := cart.UnmarshalItems(str)
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshalling cart data: %s", err.Error()))
			continue
		}

		carts = append(carts, cart.Cart{
			Items:  cartContent,
			Userid: *ct["SK"].S,
		})
	}

	return carts, nil
}

// ClearCart sets the cart for a user to an empty JSON string
func (m manager) ClearCart(userID string) error {
	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = CART SK = ID
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("CART"),
	}
	km[":id"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":payload"] = &dynamodb.AttributeValue{
		S: aws.String("{}"),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET Payload = :payload"),
	}

	_, err := dbs.UpdateItem(uii)
	return err
}

// StoreItems saves the cart items from a single user into Amazon DynamoDB
func (m manager) StoreItems(userID string, i cart.Items) error {
	payload, err := i.Marshal()
	if err != nil {
		return err
	}

	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = CART SK = ID
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("CART"),
	}
	km[":id"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":payload"] = &dynamodb.AttributeValue{
		S: aws.String(string(payload)),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET Payload = :payload"),
	}

	_, err = dbs.UpdateItem(uii)
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
