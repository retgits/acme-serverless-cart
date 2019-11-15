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

	// Create the key attributes
	userID := request.PathParameters["userid"]

	// Create a map of DynamoDB Attribute Values containing the table keys
	km := make(map[string]*dynamodb.AttributeValue)
	km[":userid"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	si := &dynamodb.ScanInput{
		TableName:                 aws.String(c.DynamoDBTable),
		ExpressionAttributeValues: km,
		FilterExpression:          aws.String("UserID = :userid"),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		errormessage := fmt.Sprintf("error querrying dynamodb: %s", err.Error())
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage
		return response, err
	}

	var cartContent cart.Cart

	if len(so.Items) == 0 {
		cartContent = cart.Cart{
			Items:  make([]cart.Item, 0),
			Userid: userID,
		}
	} else {
		str := *so.Items[0]["CartContent"].S
		cartContent, err = cart.UnmarshalCart(str)
		if err != nil {
			errormessage := fmt.Sprintf("error unmarshalling cart data: %s", err.Error())
			log.Println(errormessage)
			response.StatusCode = http.StatusInternalServerError
			response.Body = errormessage
			return response, err
		}
	}

	item, err := cart.UnmarshalItem([]byte(request.Body))
	if err != nil {
		errormessage := fmt.Sprintf("error unmarshalling item data: %s", err.Error())
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage
		return response, err
	}

	cartContent.Items = append(cartContent.Items, item)
	for idx, cci := range cartContent.Items {
		if cci.Itemid == item.Itemid {
			cartContent.Items[idx] = item
		}
	}

	cc, err := cartContent.Marshal()
	if err != nil {
		errormessage := fmt.Sprintf("error marshalling cart data: %s", err.Error())
		log.Println(errormessage)
		response.StatusCode = http.StatusInternalServerError
		response.Body = errormessage
		return response, err
	}

	km = make(map[string]*dynamodb.AttributeValue)
	km["UserID"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":cartcontent"] = &dynamodb.AttributeValue{
		S: aws.String(string(cc)),
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
