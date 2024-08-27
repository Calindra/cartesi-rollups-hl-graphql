# HL GraphQL

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

## Connecting to PostGresDB and Graphile locally

Start a PostGres instance locally using docker-compose.yml example.

```sh
docker compose up --wait postgraphile
```

When running cartesi-rollups-hl-graphql, set flag db-implementation with the value postgres

Graphile can be called using `http://localhost:5001/graphql` and you can test queries using `http://localhost:5001/graphiql`

You can change Graphile address and port using the flags graphile-url.

```sh
export POSTGRES_HOST=127.0.0.1
export POSTGRES_PORT=5432
export POSTGRES_DB=mydatabase
export POSTGRES_USER=myuser
export POSTGRES_PASSWORD=mypassword
go run . --http-address=0.0.0.0 --high-level-graphql --enable-debug --node-version v2 --db-implementation postgres
```

## Contributors

<a href="https://github.com/Calindra/cartesi-rollups-hl-graphql/graphs/contributors">
  <img src="https://contributors-img.firebaseapp.com/image?repo=calindra/cartesi-rollups-hl-graphql" />
</a>

Made with [contributors-img](https://contributors-img.firebaseapp.com).
