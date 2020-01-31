// Add item to cart
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	cart "github.com/retgits/acme-serverless-cart"
	"github.com/retgits/acme-serverless-cart/internal/datastore/dynamodb"
)

func handleError(area string, err error) (events.APIGatewayProxyResponse, error) {
	msg := fmt.Sprintf("error %s: %s", area, err.Error())
	log.Println(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       msg,
	}, err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	response := events.APIGatewayProxyResponse{}

	// Create the key attributes
	userID := request.PathParameters["userid"]

	dynamoStore := dynamodb.New()

	crt, err := cart.UnmarshalCart(request.Body)
	if err != nil {
		return handleError("unmarshalling items", err)
	}

	err = dynamoStore.StoreItems(userID, crt.Items)
	if err != nil {
		return handleError("storing items", err)
	}

	res := cart.UserIDResponse{
		UserID: userID,
	}

	payload, err := res.Marshal()
	if err != nil {
		return handleError("marshalling response", err)
	}

	headers := request.Headers
	headers["Access-Control-Allow-Origin"] = "*"

	response.StatusCode = http.StatusOK
	response.Body = string(payload)
	response.Headers = headers

	return response, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(handler)
}
