# Cart

> A cart service, because what is a shop without a cart to put stuff in?

The Cart service is part of the [ACME Fitness Serverless Shop](https://github.com/retgits/acme-serverless). The goal of this specific service is to keep track of carts and items in the different carts.

## Prerequisites

* [Go (at least Go 1.12)](https://golang.org/dl/)
* [An AWS account](https://portal.aws.amazon.com/billing/signup)
* [A Pulumi account](https://app.pulumi.com/signup)
* [A Sentry.io account](https://sentry.io) if you want to enable tracing and error reporting

## Deploying

To deploy the Cart Service you'll need a [Pulumi account](https://app.pulumi.com/signup). Once you have your Pulumi account and configured the [Pulumi CLI](https://www.pulumi.com/docs/get-started/aws/install-pulumi/), you can initialize a new stack using the Pulumi templates in the [pulumi](./pulumi) folder.

```bash
cd pulumi
pulumi stack init <your pulumi org>/acmeserverless-cart/dev
```

Pulumi is configured using a file called `Pulumi.dev.yaml`. A sample configuration is available in the Pulumi directory. You can rename [`Pulumi.dev.yaml.sample`](./pulumi/Pulumi.dev.yaml.sample) to `Pulumi.dev.yaml` and update the variables accordingly. Alternatively, you can change variables directly in the [main.go](./pulumi/main.go) file in the pulumi directory. The configuration contains:

```yaml
config:
  aws:region: us-west-2 ## The region you want to deploy to
  awsconfig:generic:
    sentrydsn: ## The DSN to connect to Sentry
    accountid: ## Your AWS Account ID
    wavefronturl: ## The URL of your Wavefront instance
    wavefronttoken: ## Your Wavefront API token
  awsconfig:tags:
    author: retgits ## The author, you...
    feature: acmeserverless
    team: vcs ## The team you're on
    version: 0.2.0 ## The version
```

To create the Pulumi stack, and create the Cart service, run `pulumi up`.

If you want to keep track of the resources in Pulumi, you can add tags to your stack as well.

```bash
pulumi stack tag set app:name acmeserverless
pulumi stack tag set app:feature acmeserverless-cart
pulumi stack tag set app:domain cart
```

## API

### `GET /cart/total/<userid>`

Get total amount in users cart

```bash
curl --request GET \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/total/dan
```

```json
{
  "carttotal": 804.5,
  "userid": "dan"
}
```

### `POST /cart/item/modify/<userid>`

Update an item in the cart of a user

```bash
curl --request POST \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/item/modify/dan \
  --header 'content-type: application/json' \
  --data '{"itemid":"sfsdsda3343", "quantity":2}'
```

To modify the item in a cart, the input needs to contain an `itemid` and the new `quantity`

```json
{"itemid":"sfsdsda3343", "quantity":2}
```

A successful update will return the userid

```json
{
  "userid": "dan"
}
```

### `POST /cart/modify/<userid>`

Modify the contents of a cart

```bash
curl --request POST \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/modify/dan \
  --header 'content-type: application/json' \
  --data '{
  "cart": [
    {
      "description": "fitband for any age - even babies",
      "itemid": "sdfsdfsfs",
      "name": "fitband",
      "price": 4.5,
      "quantity": 1
    },
    {
      "description": "the most awesome redpants in the world",
      "itemid": "sfsdsda3343",
      "name": "redpant",
      "price": 400,
      "quantity": 1
    }
  ],
  "userid": "dan"
}'
```

To replace the entire cart, or create a new cart for a user, send a cart object

```json
{
  "cart": [
    {
      "description": "fitband for any age - even babies",
      "itemid": "sdfsdfsfs",
      "name": "fitband",
      "price": 4.5,
      "quantity": 1
    }
  ],
  "userid": "dan"
}
```

A successful update will return the userid

```json
{
  "userid": "dan"
}
```

### `POST /cart/item/add/<userid>`

Add item to cart

```bash
curl --request POST \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/item/add/shri \
  --header 'content-type: application/json' \
  --data '{"itemid":"xyz", "quantity":3}'
```

To add the item in a cart, the input needs to contain an `itemid` and the `quantity`

```json
{"itemid":"xyz", "quantity":3}
```

A successful update will return the userid

```json
{
  "userid": "shri"
}
```

### `GET /cart/items/total/<userid>`

Get the total number of items in a cart

```bash
curl --request GET \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/items/total/shri
```

```json
{
  "cartitemtotal": 5.0,
  "userid": "shri"
}
```

### `GET /cart/clear/<userid>`

Clear all items from the cart

```bash
curl --request GET \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/clear/dan
```

```text
<no payload returned>
```

### `GET /cart/items/<userid>`

Get all items in a cart

```bash
curl --request GET \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/items/dan
```

```json
{
  "cart": [
    {
      "description": "fitband for any age - even babies",
      "itemid": "sdfsdfsfs",
      "name": "fitband",
      "price": 4.5,
      "quantity": 1
    },
    {
      "description": "the most awesome redpants in the world",
      "itemid": "sfsdsda3343",
      "name": "redpant",
      "price": 400,
      "quantity": 1
    }
  ],
  "userid": "dan"
}
```

### `GET /cart/all`

Get all the carts

```bash
curl --request GET \
  --url https://<id>.execute-api.us-west-2.amazonaws.com/Prod/cart/all
```

```json
[
    {
        "cart": [
            {
                "description": "fitband for any age - even babies",
                "itemid": "sdfsdfsfs",
                "name": "fitband",
                "price": 4.5,
                "quantity": 1
            },
            {
                "description": "the most awesome redpants in the world",
                "itemid": "sfsdsda3343",
                "name": "redpant",
                "price": 400,
                "quantity": 1
            }
        ],
        "id": "shri"
    }
]
```

## Building for Google Cloud Run

If you have Docker installed locally, you can use `docker build` to create a container which can be used to try out the cart service locally and for Google Cloud Run.

To build your container image using Docker:

Run the command:

```bash
VERSION=`git describe --tags --always --dirty="-dev"`
docker build -f ./cmd/cloudrun-cart-http/Dockerfile . -t gcr.io/[PROJECT-ID]/cart:$VERSION
```

Replace `[PROJECT-ID]` with your Google Cloud project ID

If you have not yet configured Docker to use the gcloud command-line tool to authenticate requests to Container Registry, do so now using the command:

```bash
gcloud auth configure-docker
```

You need to do this before you can push or pull images using Docker. You only need to do it once.

Push the container image to Container Registry:

```bash
docker push gcr.io/[PROJECT-ID]/cart:$VERSION
```

The container relies on the environment variables:

* SENTRY_DSN: The DSN to connect to Sentry
* K_SERVICE: The name of the service (in Google Cloud Run this variable is automatically set, defaults to `cart` if not set)
* VERSION: The version you're running (will default to `dev` if not set)
* PORT: The port number the service will listen on (will default to `8080` if not set)
* STAGE: The environment in which you're running
* WAVEFRONT_TOKEN: The token to connect to Wavefront
* WAVEFRONT_URL: The URL to connect to Wavefront (will default to `debug` if not set)
* MONGO_USERNAME: The username to connect to MongoDB
* MONGO_PASSWORD: The password to connect to MongoDB
* MONGO_HOSTNAME: The hostname of the MongoDB server
* MONGO_PORT: The port number of the MongoDB server

A `docker run`, with all options, is:

```bash
docker run --rm -it -p 8080:8080 -e SENTRY_DSN=abcd -e K_SERVICE=cart \
  -e VERSION=$VERSION -e PORT=8080 -e STAGE=dev -e WAVEFRONT_URL=https://my-url.wavefront.com \
  -e WAVEFRONT_TOKEN=efgh -e MONGO_USERNAME=admin -e MONGO_PASSWORD=admin \
  -e MONGO_HOSTNAME=localhost -e MONGO_PORT=27017 gcr.io/[PROJECT-ID]/cart:$VERSION
```

Replace `[PROJECT-ID]` with your Google Cloud project ID

## Troubleshooting

In case the API Gateway responds with `{"message":"Forbidden"}`, there is likely an issue with the deployment of the API Gateway. To solve this problem, you can use the AWS CLI. To confirm this, run `aws apigateway get-deployments --rest-api-id <rest-api-id>`. If that returns no deployments, you can create a deployment for the *prod* stage with `aws apigateway create-deployment --rest-api-id <rest-api-id> --stage-name prod --stage-description 'Prod Stage' --description 'deployment to the prod stage'`.

## Contributing

[Pull requests](https://github.com/retgits/acme-serverless-cart/pulls) are welcome. For major changes, please open [an issue](https://github.com/retgits/acme-serverless-cart/issues) first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

See the [LICENSE](./LICENSE) file in the repository
