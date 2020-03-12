# Cart

> A cart service, because what is a shop without a cart to put stuff in?

The Cart service is part of the [ACME Fitness Serverless Shop](https://github.com/retgits/acme-serverless). The goal of this specific service is to keep track of carts and items in the different carts.

## Prerequisites

* [Go (at least Go 1.12)](https://golang.org/dl/)
* [An AWS account](https://portal.aws.amazon.com/billing/signup)
* [A Pulumi account](https://app.pulumi.com/signup)
* [A Sentry.io account](https://sentry.io) if you want to enable tracing and error reporting

## Deploying

To deploy the Payment Service you'll need a [Pulumi account](https://app.pulumi.com/signup). Once you have your Pulumi account and configured the [Pulumi CLI](https://www.pulumi.com/docs/get-started/aws/install-pulumi/), you can initialize a new stack using the Pulumi templates in the [pulumi](./pulumi) folder.

```bash
cd pulumi
pulumi stack init <your pulumi org>/acmeserverless-cart/dev
```

You'll need to create a [Pulumi.dev.yaml](./pulumi/Pulumi.dev.yaml) file that will contain all configuration data to deploy the app:

```yaml
config:
  aws:region: us-west-2 ## The region you want to deploy to
  awsconfig:lambda:
    dynamoarn: ## The ARN to the DynamoDB table
    sentrydsn: ## The DSN to connect to Sentry
    region: ## The region you want to deploy to
    accountid: ## Your AWS sccount ID
  awsconfig:tags:
    author: retgits ## The author, you...
    feature: acmeserverless
    team: vcs ## The team you're on
    version: 0.1.0 ## The version
```

To create the Pulumi stack, and create the Payment service, run `pulumi up`.

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

## Using Make

The Makefiles and CloudFormation templates can be found in the [acme-serverless](https://github.com/retgits/acme-serverless/tree/master/deploy/cloudformation/cart) repository

## Using Mage

If you want to "go all Go" (_pun intended_) and write plain-old go functions to build and deploy, you can use [Mage](https://magefile.org/). Mage is a make/rake-like build tool using Go so Mage automatically uses the functions you create as Makefile-like runnable targets.

The Magefile can be found in the [acme-serverless](https://github.com/retgits/acme-serverless/tree/master/deploy/mage) repository

## Contributing

[Pull requests](https://github.com/retgits/acme-serverless-cart/pulls) are welcome. For major changes, please open [an issue](https://github.com/retgits/acme-serverless-cart/issues) first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

See the [LICENSE](./LICENSE) file in the repository
