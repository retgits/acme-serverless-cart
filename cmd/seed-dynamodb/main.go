package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	cart "github.com/retgits/acme-serverless-cart"
	"github.com/retgits/acme-serverless-cart/internal/datastore/dynamodb"
)

func main() {
	os.Setenv("REGION", "us-west-2")
	os.Setenv("TABLE", "Cart")

	data, err := ioutil.ReadFile("./data.json")
	if err != nil {
		log.Println(err)
	}

	var carts cart.Carts

	err = json.Unmarshal(data, &carts)
	if err != nil {
		log.Println(err)
	}

	dynamoStore := dynamodb.New()

	for _, crt := range carts {
		err = dynamoStore.StoreItems(crt.Userid, crt.Items)
	}
}
