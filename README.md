# Cart

> A cart service, because what is a shop without a cart to put stuff in?

The Cart service is part of the [ACME Fitness Serverless Shop](https://github.com/vmwarecloudadvocacy/acme_fitness_demo). The goal of this specific service is to keep track of carts and items in the different carts.

## Prerequisites

* [Go (at least Go 1.12)](https://golang.org/dl/)
* [An AWS Account](https://portal.aws.amazon.com/billing/signup)
* The _vuln_ targets for Make and Mage rely on the [Snyk](http://snyk.io/) CLI

## Eventing Options

The cart service has Lambdas triggered by [Amazon API Gateway](https://aws.amazon.com/api-gateway/)

## Data Stores

The cart service supports the following data stores:

* [Amazon DynamoDB](https://aws.amazon.com/dynamodb/). The table can be created using the makefile in [create-dynamodb](./cmd/create-dynamodb).

## Using Amazon API Gateway

### Prerequisites for Amazon API Gateway

* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed and configured

### Build and deploy for Amazon API Gateway

Clone this repository

```bash
git clone https://github.com/retgits/acme-serverless-cart
cd acme-serverless-cart
```

Get the Go Module dependencies

```bash
go get ./...
```

Switch directories to any of the Lambda folders

```bash
cd ./cmd/lambda-cart-<name>
```

Use make to deploy

```bash
make build
make deploy
```

### Testing Amazon API Gateway

After the deployment you'll see the URL to which you can send the below mentioned API requests

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
{
  "all carts": [
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
}
```

## Contributing

[Pull requests](https://github.com/retgits/acme-serverless-cart/pulls) are welcome. For major changes, please open [an issue](https://github.com/retgits/acme-serverless-cart/issues) first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

See the [LICENSE](./LICENSE) file in the repository