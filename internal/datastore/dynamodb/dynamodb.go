package dynamodb

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	cart "github.com/retgits/acme-serverless-cart"
	"github.com/retgits/acme-serverless-cart/internal/datastore"
)

type manager struct{}

func New() datastore.Manager {
	return manager{}
}

func (m manager) GetItems(userID string) (cart.Items, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	km := make(map[string]*dynamodb.AttributeValue)
	km[":userid"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	si := &dynamodb.ScanInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		ExpressionAttributeValues: km,
		FilterExpression:          aws.String("UserID = :userid"),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		return nil, err
	}

	if so.Items[0]["CartContent"].S == nil {
		return nil, fmt.Errorf("no items found for user")
	}

	str := *so.Items[0]["CartContent"].S
	return cart.UnmarshalItems(str)
}

func (m manager) AddItem(userID string, i cart.Item) error {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

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
	km["UserID"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":cartcontent"] = &dynamodb.AttributeValue{
		S: aws.String(string(cc)),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET CartContent = :cartcontent"),
	}

	_, err = dbs.UpdateItem(uii)
	return err
}

func (m manager) AllCarts() (cart.Carts, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	si := &dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("TABLE")),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		return nil, err
	}

	if len(so.Items) == 0 {
		return nil, fmt.Errorf("no item data found")
	}

	carts := make(cart.Carts, 0)

	for _, ct := range so.Items {
		str := *ct["CartContent"].S

		cartContent, err := cart.UnmarshalItems(str)
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshalling cart data: %s", err.Error()))
			continue
		}

		carts = append(carts, cart.Cart{
			Items:  cartContent,
			Userid: *ct["UserID"].S,
		})
	}

	return carts, nil
}

func (m manager) ClearCart(userID string) error {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	km := make(map[string]*dynamodb.AttributeValue)
	km["UserID"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":cartcontent"] = &dynamodb.AttributeValue{
		S: aws.String(" "),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET CartContent = :cartcontent"),
	}

	_, err := dbs.UpdateItem(uii)
	return err
}

func (m manager) StoreItems(userID string, i cart.Items) error {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	payload, err := i.Marshal()
	if err != nil {
		return err
	}

	km := make(map[string]*dynamodb.AttributeValue)
	km["UserID"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":cartcontent"] = &dynamodb.AttributeValue{
		S: aws.String(string(payload)),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET CartContent = :cartcontent"),
	}

	_, err = dbs.UpdateItem(uii)
	return err
}

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
