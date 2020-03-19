package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pulumi/pulumi-aws/sdk/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
	"github.com/pulumi/pulumi/sdk/go/pulumi/config"
	"github.com/retgits/pulumi-helpers/builder"
	gw "github.com/retgits/pulumi-helpers/gateway"
	"github.com/retgits/pulumi-helpers/sampolicies"
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

// GenericConfig contains the key-value pairs for the configuration of AWS in this stack
type GenericConfig struct {
	// The AWS region used
	Region string

	// The DSN used to connect to Sentry
	SentryDSN string `json:"sentrydsn"`

	// The AWS AccountID to use
	AccountID string `json:"accountid"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the region
		region, found := ctx.GetConfig("aws:region")
		if !found {
			return fmt.Errorf("region not found")
		}

		// Read the configuration data from Pulumi.<stack>.yaml
		conf := config.New(ctx, "awsconfig")

		// Create a new Tags object with the data from the configuration
		var tags Tags
		conf.RequireObject("tags", &tags)

		// Create a new GenericConfig object with the data from the configuration
		var genericConfig GenericConfig
		conf.RequireObject("generic", &genericConfig)
		genericConfig.Region = region

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

		// Build the functions
		for _, fnName := range functions {
			fnFolder := path.Join(wd, "..", "cmd", fnName)
			buildFactory := builder.NewFactory().WithFolder(fnFolder)
			buildFactory.MustBuild()
			buildFactory.MustZip()
		}

		// Create a factory to get policies from
		iamFactory := sampolicies.NewFactory().WithAccountID(genericConfig.AccountID).WithPartition("aws").WithRegion(genericConfig.Region)

		// Lookup the DynamoDB table
		dynamoTable, err := dynamodb.LookupTable(ctx, &dynamodb.LookupTableArgs{
			Name: fmt.Sprintf("%s-acmeserverless-dynamodb", ctx.Stack()),
		})

		// dynamoPolicy is a policy template, derived from AWS SAM, to allow apps
		// to connect to and execute command on Amazon DynamoDB
		iamFactory.ClearPolicies()
		iamFactory.AddDynamoDBCrudPolicy(dynamoTable.Name)
		dynamoPolicy, err := iamFactory.GetPolicyStatement()
		if err != nil {
			return err
		}

		roles := make(map[string]*iam.Role)

		// Create a new IAM role for each Lambda function
		for _, function := range functions {
			// Give the role the ability to run on AWS Lambda
			roleArgs := &iam.RoleArgs{
				AssumeRolePolicy: pulumi.String(sampolicies.AssumeRoleLambda()),
				Description:      pulumi.String(fmt.Sprintf("Role for the Cart Service (%s) of the ACME Serverless Fitness Shop", function)),
				Tags:             pulumi.Map(tagMap),
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
				Policy: pulumi.String(dynamoPolicy),
			})
			if err != nil {
				return err
			}

			ctx.Export(fmt.Sprintf("%s-role::Arn", function), role.Arn)
			roles[function] = role
		}

		// All functions will have the same environment variables, with the exception
		// of the function name
		variables := make(map[string]pulumi.StringInput)
		variables["REGION"] = pulumi.String(genericConfig.Region)
		variables["SENTRY_DSN"] = pulumi.String(genericConfig.SentryDSN)
		variables["VERSION"] = tags.Version
		variables["STAGE"] = pulumi.String(ctx.Stack())
		variables["TABLE"] = pulumi.String(dynamoTable.Name)

		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-additem", ctx.Stack()))
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

		cartAddItemFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-additem", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-additem::Arn", cartAddItemFunction.Arn)

		// Create the All function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-all", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		cartAllFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-all", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-all::Arn", cartAllFunction.Arn)

		// Create the Clear function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-clear", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		cartClearFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-clear", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-clear::Arn", cartClearFunction.Arn)

		// Create the ItemModify function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-itemmodify", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}
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

		cartItemModifyFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-itemmodify", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-itemmodify::Arn", cartItemModifyFunction.Arn)

		// Create the ItemTotal function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-itemtotal", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		cartItemTotalFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-itemtotal", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-itemtotal::Arn", cartItemTotalFunction.Arn)

		// Create the Modify function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-modify", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-modify", ctx.Stack()))

		cartModifyFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-modify", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-modify::Arn", cartModifyFunction.Arn)

		// Create the Total function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-total", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-total", ctx.Stack()))

		cartTotalFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-total", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-total::Arn", cartTotalFunction.Arn)

		// Create the User function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-cart-user", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		cartUserFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-cart-user", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-cart-user::Arn", cartUserFunction.Arn)

		// Create the API Gateway Policy
		iamFactory.ClearPolicies()
		iamFactory.AddAssumeRoleLambda()
		iamFactory.AddExecuteAPI()
		policies, err := iamFactory.GetPolicyStatement()
		if err != nil {
			return err
		}

		// Read the OpenAPI specification
		bytes, err := ioutil.ReadFile("../api/openapi.json")
		if err != nil {
			return err
		}

		// Create an API Gateway
		gateway, err := apigateway.NewRestApi(ctx, "CartService", &apigateway.RestApiArgs{
			Name:        pulumi.String("CartService"),
			Description: pulumi.String("ACME Serverless Fitness Shop - Cart"),
			Tags:        pulumi.Map(tagMap),
			Policy:      pulumi.String(policies),
			Body:        pulumi.StringPtr(string(bytes)),
		})
		if err != nil {
			return err
		}

		gatewayURL := gateway.ID().ToStringOutput().ApplyString(func(id string) string {
			resource := gw.MustGetGatewayResource(ctx, id, "/cart/item/add/{userid}")

			apigateway.NewIntegration(ctx, "AddItemAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartAddItemFunction.InvokeArn,
			})

			_, err = lambda.NewPermission(ctx, "AddItemAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartAddItemFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/item/add/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/all")

			_, err = apigateway.NewIntegration(ctx, "AllCartsAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartAllFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "AllCartsAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartAllFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/all", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/clear/{userid}")

			_, err = apigateway.NewIntegration(ctx, "ClearCartAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartClearFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "ClearCartAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartClearFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/clear/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/item/modify/{userid}")

			_, err = apigateway.NewIntegration(ctx, "ItemModifyAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartItemModifyFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "ItemModifyAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartItemModifyFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/item/modify/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/items/total/{userid}")

			_, err = apigateway.NewIntegration(ctx, "ItemTotalAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartItemTotalFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "ItemTotalAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartItemTotalFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/items/total/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/modify/{userid}")

			_, err = apigateway.NewIntegration(ctx, "CartModifyAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartModifyFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "CartModifyAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartModifyFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/cart/modify/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/total/{userid}")

			_, err = apigateway.NewIntegration(ctx, "CartTotalAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartTotalFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "CartTotalAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartTotalFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/total/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/cart/items/{userid}")

			_, err = apigateway.NewIntegration(ctx, "CartUserTotalAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   cartUserFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "CartUserTotalAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  cartUserFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/cart/items/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}
			return fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/prod/", id, genericConfig.Region)
		})

		ctx.Export("Gateway::URL", gatewayURL)

		return nil
	})
}
