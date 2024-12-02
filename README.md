# Cartesi's HL GraphQL

![CI](https://github.com/Calindra/cartesi-rollups-hl-graphql/actions/workflows/ci.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Calindra/cartesi-rollups-hl-graphql)](https://goreportcard.com/report/github.com/Calindra/cartesi-rollups-hl-graphql)

[Technical Vision Forum Discussion](https://governance.cartesi.io/t/convenience-layer-for-voucher-management-on-cartesi/401)

[Internal docs](./docs/convenience.md)

## Description

Exposes the High Level GraphQL reader API in the endpoint `http://127.0.0.1:8080/graphql`.
You may access this address to use the GraphQL interactive playground in your web browser.
You can also make POST requests directly to the GraphQL API.
For instance, the command below gets the number of inputs.

```sh
QUERY='query { inputs { totalCount } }'; \
curl \
    -X POST \
    -H 'Content-Type: application/json' \
    -d "{\"query\": \"$QUERY\"}" \
    http://127.0.0.1:8080/graphql
```

## Connecting to Postgres locally

Start a Postgres instance locally using docker compose.

```sh
make up-db-raw
```

New configuration

```sh
export POSTGRES_GRAPHQL_DB_URL="postgres://postgres:password@localhost:5432/hlgraphql?sslmode=disable"
export POSTGRES_NODE_DB_URL="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
./cartesi-rollups-hl-graphql
```

Old configuration

When running cartesi-rollups-hl-graphql, set flag db-implementation with the value postgres

```sh
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=hlgraphql
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=password
export POSTGRES_NODE_DB_URL="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
./cartesi-rollups-hl-graphql --raw-enabled --graphile-disable-sync --db-implementation=postgres
```

## Contributors

<a href="https://github.com/Calindra/cartesi-rollups-hl-graphql/graphs/contributors">
  <img src="https://contributors-img.firebaseapp.com/image?repo=calindra/cartesi-rollups-hl-graphql" />
</a>

Made with [contributors-img](https://contributors-img.firebaseapp.com).
