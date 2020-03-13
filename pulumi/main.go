package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
	"github.com/pulumi/pulumi/sdk/go/pulumi/config"
)

const (
	// The shell to use
	shell = "sh"

	// The flag for the shell to read commands from a string
	shellFlag = "-c"
)

// Tags are key-value pairs to apply to the resources created by this stack
type Tags struct {
	// Author is the person who created the code, or performed the deployment
	Author pulumi.String

	// Feature is the project that this resource belongs to
	Feature pulumi.String

	// Team is the team that is responsible to manage this resource
	Team pulumi.String

	// Version is the version of the code for this resource
	Version pulumi.String
}

// LambdaConfig contains the key-value pairs for the configuration of AWS Lambda in this stack
type LambdaConfig struct {
	// The DSN used to connect to Sentry
	SentryDSN string `json:"sentrydsn"`

	// The ARN for the DynamoDB table
	DynamoARN string `json:"dynamoarn"`

	// The AWS region used
	Region string `json:"region"`

	// The AWS AccountID used
	AccountID string `json:"accountid"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Read the configuration data from Pulumi.<stack>.yaml
		conf := config.New(ctx, "awsconfig")

		// Create a new Tags object with the data from the configuration
		var tags Tags
		conf.RequireObject("tags", &tags)

		// Create a new DynamoConfig object with the data from the configuration
		var lambdaConfig LambdaConfig
		conf.RequireObject("lambda", &lambdaConfig)

		// Create a map[string]pulumi.Input of the tags
		// the first four tags come from the configuration file
		// the last two are derived from this deployment
		tagMap := make(map[string]pulumi.Input)
		tagMap["Author"] = tags.Author
		tagMap["Feature"] = tags.Feature
		tagMap["Team"] = tags.Team
		tagMap["Version"] = tags.Version
		tagMap["ManagedBy"] = pulumi.String("Pulumi")
		tagMap["Stage"] = pulumi.String(ctx.Stack())

		// functions are the functions that need to be deployed
		functions := []string{
			"lambda-cart-additem",
			"lambda-cart-all",
			"lambda-cart-clear",
			"lambda-cart-itemmodify",
			"lambda-cart-itemtotal",
			"lambda-cart-modify",
			"lambda-cart-total",
			"lambda-cart-user",
		}

		// Compile and zip the AWS Lambda functions
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		for _, fnName := range functions {
			// Find the working folder
			fnFolder := path.Join(wd, "..", "cmd", fnName)

			// Run go build
			if err := run(fnFolder, "GOOS=linux GOARCH=amd64 go build"); err != nil {
				fmt.Printf("Error building code: %s", err.Error())
				os.Exit(1)
			}

			// Zip up the binary
			if err := run(fnFolder, fmt.Sprintf("zip ./%s.zip ./%s", fnName, fnName)); err != nil {
				fmt.Printf("Error creating zipfile: %s", err.Error())
				os.Exit(1)
			}
		}

		// Create an API Gateway
		gateway, err := apigateway.NewRestApi(ctx, "CartService", &apigateway.RestApiArgs{
			Name:        pulumi.String("CartService"),
			Description: pulumi.String("ACME Serverless Fitness Shop - Cart"),
			Tags:        pulumi.Map(tagMap),
			Policy:      pulumi.String(`{ "Version": "2012-10-17", "Statement": [ { "Action": "sts:AssumeRole", "Principal": { "Service": "lambda.amazonaws.com" }, "Effect": "Allow", "Sid": "" },{ "Action": "execute-api:Invoke", "Resource":"execute-api:/*", "Principal": "*", "Effect": "Allow", "Sid": "" } ] }`),
		})
		if err != nil {
			return err
		}

		// Create the parent resources in the API Gateway
		cartResource, err := apigateway.NewResource(ctx, "CartAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("cart"),
			ParentId: gateway.RootResourceId,
		})
		if err != nil {
			return err
		}

		itemResource, err := apigateway.NewResource(ctx, "ItemAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("item"),
			ParentId: cartResource.ID(),
		})
		if err != nil {
			return err
		}

		clearResource, err := apigateway.NewResource(ctx, "ClearAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("clear"),
			ParentId: cartResource.ID(),
		})
		if err != nil {
			return err
		}

		cartModifyResource, err := apigateway.NewResource(ctx, "CartModifyAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("modify"),
			ParentId: cartResource.ID(),
		})
		if err != nil {
			return err
		}

		itemAddResource, err := apigateway.NewResource(ctx, "ItemAddAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("add"),
			ParentId: itemResource.ID(),
		})
		if err != nil {
			return err
		}

		itemModifyResource, err := apigateway.NewResource(ctx, "ItemModifyAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("modify"),
			ParentId: itemResource.ID(),
		})
		if err != nil {
			return err
		}

		itemsResource, err := apigateway.NewResource(ctx, "ItemsAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("items"),
			ParentId: cartResource.ID(),
		})
		if err != nil {
			return err
		}

		totalResource, err := apigateway.NewResource(ctx, "TotalAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("total"),
			ParentId: cartResource.ID(),
		})
		if err != nil {
			return err
		}

		itemsTotalResource, err := apigateway.NewResource(ctx, "ItemsTotalAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("total"),
			ParentId: itemsResource.ID(),
		})
		if err != nil {
			return err
		}

		// dynamoCRUDPolicyString is a policy template, derived from AWS SAM, to allow apps
		// to connect to and execute command on Amazon DynamoDB
		dynamoCRUDPolicyString := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Action": [
						"dynamodb:GetItem",
						"dynamodb:DeleteItem",
						"dynamodb:PutItem",
						"dynamodb:Scan",
						"dynamodb:Query",
						"dynamodb:UpdateItem",
						"dynamodb:BatchWriteItem",
						"dynamodb:BatchGetItem",
						"dynamodb:DescribeTable",
						"dynamodb:ConditionCheckItem"
					],
					"Effect": "Allow",
					"Resource": "%s"
				}
			]
		}`, lambdaConfig.DynamoARN)

		roles := make(map[string]*iam.Role)

		// Create a new IAM role for each Lambda function
		for _, function := range functions {
			// Give the role the ability to run on AWS Lambda
			roleArgs := &iam.RoleArgs{
				AssumeRolePolicy: pulumi.String(`{
					"Version": "2012-10-17",
					"Statement": [
					{
						"Action": "sts:AssumeRole",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						},
						"Effect": "Allow",
						"Sid": ""
					}
					]
				}`),
				Description: pulumi.String(fmt.Sprintf("Role for the Cart Service (%s) of the ACME Serverless Fitness Shop", function)),
				Tags:        pulumi.Map(tagMap),
			}

			role, err := iam.NewRole(ctx, fmt.Sprintf("ACMEServerlessCartRole-%s", function), roleArgs)
			if err != nil {
				return err
			}

			// Attach the AWSLambdaBasicExecutionRole so the function can create Log groups in CloudWatch
			_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("AWSLambdaBasicExecutionRole-%s", function), &iam.RolePolicyAttachmentArgs{
				PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
				Role:      role.Name,
			})
			if err != nil {
				return err
			}

			// Add the DynamoDB policy
			_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("ACMEServerlessCartPolicy-%s", function), &iam.RolePolicyArgs{
				Name:   pulumi.String(fmt.Sprintf("ACMEServerlessCartPolicy-%s", function)),
				Role:   role.Name,
				Policy: pulumi.String(dynamoCRUDPolicyString),
			})
			if err != nil {
				return err
			}

			ctx.Export(fmt.Sprintf("%s-role::Arn", function), role.Arn)
			roles[function] = role
		}

		// All functions will have the same environment variables
		variables := make(map[string]pulumi.StringInput)
		variables["REGION"] = pulumi.String(lambdaConfig.Region)
		variables["SENTRY_DSN"] = pulumi.String(lambdaConfig.SentryDSN)
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-payment", ctx.Stack()))
		variables["VERSION"] = tags.Version
		variables["STAGE"] = pulumi.String(ctx.Stack())
		parts := strings.Split(lambdaConfig.DynamoARN, "/")
		variables["TABLE"] = pulumi.String(parts[1])

		environment := lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

		// Create the AddItem function
		functionArgs := &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to add an item to a cart"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-additem", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-additem"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-additem/lambda-cart-additem.zip"),
			Role:        roles["lambda-cart-additem"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-additem", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err := apigateway.NewResource(ctx, "AddItemAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: itemAddResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "AddItemAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "AddItemAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "AddItemAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/item/add/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-additem::Arn", function.Arn)

		// Create the All function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to get all carts from DynamoDB"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-all", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-all"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-all/lambda-cart-all.zip"),
			Role:        roles["lambda-cart-all"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-all", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "AllCartsAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("all"),
			ParentId: cartResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "AllCartsAPIGetMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "AllCartsAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "AllCartsAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/all", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-all::Arn", function.Arn)

		// Create the Clear function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to clear the cart of a user"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-clear", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-clear"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-clear/lambda-cart-clear.zip"),
			Role:        roles["lambda-cart-clear"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-clear", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "ClearCartAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: clearResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "ClearCartAPIGetMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "ClearCartAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "ClearCartAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/clear/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-clear::Arn", function.Arn)

		// Create the ItemModify function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to modify an item to a cart"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-itemmodify", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-itemmodify"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-itemmodify/lambda-cart-itemmodify.zip"),
			Role:        roles["lambda-cart-itemmodify"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-itemmodify", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "ItemModifyAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: itemModifyResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "ItemModifyAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "ItemModifyAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "ItemModifyAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/item/modify/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-itemmodify::Arn", function.Arn)

		// Create the ItemTotal function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to get the total number of items in a cart from DynamoDB based on the userID"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-itemtotal", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-itemtotal"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-itemtotal/lambda-cart-itemtotal.zip"),
			Role:        roles["lambda-cart-itemtotal"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-itemtotal", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "ItemTotalAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: itemsTotalResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "ItemTotalAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "ItemTotalAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "ItemTotalAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/items/total/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-itemtotal::Arn", function.Arn)

		// Create the Modify function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to modify a cart"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-modify", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-modify"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-modify/lambda-cart-modify.zip"),
			Role:        roles["lambda-cart-modify"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-modify", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "CartModifyAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: cartModifyResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "CartModifyAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "CartModifyAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "CartModifyAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/modify/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-modify::Arn", function.Arn)

		// Create the Total function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to get the total value of items in a cart from DynamoDB based on the userID"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-total", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-total"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-total/lambda-cart-total.zip"),
			Role:        roles["lambda-cart-total"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-total", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "CartTotalAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: totalResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "CartTotalAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "CartTotalAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "CartTotalAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/total/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-total::Arn", function.Arn)

		// Create the User function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to get cart items from DynamoDB based on the userID"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-cart-user", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-cart-user"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-cart-user/lambda-cart-user.zip"),
			Role:        roles["lambda-cart-user"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-user", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "CartUserTotalAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{userid}"),
			ParentId: itemsResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "CartUserTotalAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "CartUserTotalAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "CartUserTotalAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/items/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-user::Arn", function.Arn)

		return nil
	})
}

// run creates a Cmd struct to execute the named program with the given arguments.
// After that, it starts the specified command and waits for it to complete.
func run(folder string, args string) error {
	cmd := exec.Command(shell, shellFlag, args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = folder
	return cmd.Run()
}
