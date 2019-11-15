// Get all the carts
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/kelseyhightower/envconfig"
	"github.com/retgits/cart"
	wflambda "github.com/wavefronthq/wavefront-lambda-go"
)

var wfAgent = wflambda.NewWavefrontAgent(&wflambda.WavefrontConfig{})

// config is the struct that is used to keep track of all environment variables
type config struct {
	AWSRegion     string `required:"true" split_words:"true" envconfig:"REGION"`
	DynamoDBTable string `required:"true" split_words:"true" envconfig:"TABLENAME"`
}

var c config

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	response := events.APIGatewayProxyResponse{}

	// Get configuration set using environment variables
	err := envconfig.Process("", &c)
	if err != nil {
		errormessage := fmt.Sprintf("error starting function: %s", err.Error())
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage
		return response, err
	}

	// Create an AWS session
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(c.AWSRegion),
	}))

	// Create a DynamoDB session
	dbs := dynamodb.New(awsSession)

	si := &dynamodb.ScanInput{
		TableName: aws.String(c.DynamoDBTable),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		errormessage := fmt.Sprintf("error querrying dynamodb: %s", err.Error())
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage
		return response, err
	}

	if len(so.Items) == 0 {
		errormessage := fmt.Errorf("no cart data found")
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage.Error()
		return response, err
	}

	carts := make(cart.Carts, len(so.Items))

	for idx, ct := range so.Items {
		str := *ct["CartContent"].S
		cartContent, err := cart.UnmarshalCart(str)
		if err != nil {
			errormessage := fmt.Sprintf("error unmarshalling cart data: %s", err.Error())
			log.Println(errormessage)
			break
		}
		carts[idx] = cartContent
	}

	pl, _ := carts.Marshal()

	response.StatusCode = http.StatusOK
	response.Body = string(pl)

	return response, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(wfAgent.WrapHandler(handler))
}
