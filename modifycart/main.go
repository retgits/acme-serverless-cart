// Add item to cart
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

	// Create the key attributes
	userID := request.PathParameters["userid"]

	km := make(map[string]*dynamodb.AttributeValue)
	km["UserID"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":cartcontent"] = &dynamodb.AttributeValue{
		S: aws.String(request.Body),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(c.DynamoDBTable),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET CartContent = :cartcontent"),
	}

	_, err = dbs.UpdateItem(uii)
	if err != nil {
		errormessage := fmt.Sprintf("error updating dynamodb: %s", err.Error())
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage
		return response, err
	}

	response.StatusCode = http.StatusOK
	response.Body = userID

	return response, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(wfAgent.WrapHandler(handler))
}
