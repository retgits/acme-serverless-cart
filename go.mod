module github.com/retgits/acme-serverless-cart

replace github.com/wavefronthq/wavefront-lambda-go => github.com/retgits/wavefront-lambda-go v0.0.0-20200406192713-6ff30b7e488c

go 1.13

require (
	github.com/aws/aws-lambda-go v1.16.0
	github.com/aws/aws-sdk-go v1.30.7
	github.com/getsentry/sentry-go v0.5.1
	github.com/pulumi/pulumi v1.14.1
	github.com/pulumi/pulumi-aws v1.31.0
	github.com/pulumi/pulumi/sdk v1.14.1
	github.com/retgits/pulumi-helpers v0.1.7
	github.com/wavefronthq/wavefront-lambda-go v0.0.0-20190812171804-d9475d6695cc
)
