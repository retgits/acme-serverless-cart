// Add item to cart
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/getsentry/sentry-go"
	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/retgits/acme-serverless-cart/internal/datastore/dynamodb"
	wflambda "github.com/wavefronthq/wavefront-lambda-go"
)

// handler handles the API Gateway events and returns an error if anything goes wrong.
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Initiialize a connection to Sentry to capture errors and traces
	sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("SENTRY_DSN"),
		Transport: &sentry.HTTPSyncTransport{
			Timeout: time.Second * 3,
		},
		ServerName:  os.Getenv("FUNCTION_NAME"),
		Release:     os.Getenv("VERSION"),
		Environment: os.Getenv("STAGE"),
	})

	// Create headers if they don't exist and add
	// the CORS required headers, otherwise the response
	// will not be accepted by browsers.
	headers := request.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Access-Control-Allow-Origin"] = "*"

	// Create the key attributes
	userID := request.PathParameters["userid"]

	dynamoStore := dynamodb.New()

	cartItems, err := dynamoStore.GetItems(userID)
	if err != nil {
		return handleError("getting items", headers, err)
	}

	item, err := acmeserverless.UnmarshalItem([]byte(request.Body))
	if err != nil {
		return handleError("unmarshaling item data", headers, err)
	}

	for idx, cci := range cartItems {
		if cci.ItemID == item.ItemID {
			cartItems[idx] = item
		}
	}

	err = dynamoStore.StoreItems(userID, cartItems)
	if err != nil {
		return handleError("storing modified data", headers, err)
	}

	res := acmeserverless.UserIDResponse{
		UserID: userID,
	}

	payload, err := res.Marshal()
	if err != nil {
		return handleError("marshalling response", headers, err)
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(payload),
		Headers:    headers,
	}

	return response, nil
}

// handleError takes the activity where the error occured and the error object and sends a message to sentry.
// The original error, together with the appropriate API Gateway Proxy Response, is returned so it can be thrown.
func handleError(area string, headers map[string]string, err error) (events.APIGatewayProxyResponse, error) {
	sentry.CaptureException(fmt.Errorf("error %s: %s", area, err.Error()))
	msg := fmt.Sprintf("error %s: %s", area, err.Error())
	log.Println(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusBadRequest,
		Body:       msg,
		Headers:    headers,
	}, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(wflambda.Wrapper(handler))
}
