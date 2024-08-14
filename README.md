# HL GraphQL

![CI](https://github.com/Calindra/cartesi-rollups-hl-graphql/actions/workflows/ci.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Calindra/cartesi-rollups-hl-graphql)](https://goreportcard.com/report/github.com/Calindra/cartesi-rollups-hl-graphql)

Exposes the GraphQL reader API in the endpoint `http://127.0.0.1:8080/graphql`.
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

Start a PostGres instance locally, "cd" to db folder and use docker-compose.yml example.
Set PostGres connection details using environment variables

```env
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=mydatabase
export POSTGRES_USER=myuser
export POSTGRES_PASSWORD=mypassword
```

When running cartesi-rollups-hl-graphql, set flag db-implementation with the value postgres

Graphile can be called using `http://localhost:5001/graphql` and you can test queries using `http://localhost:5001/graphiql`

You can change Graphile address and port using the flags graphile-url.

```sh
./cartesi-rollups-hl-graphql --graphile-url http://mygraphileaddress:5034
```

## Contributors

<a href="https://github.com/Calindra/cartesi-rollups-hl-graphql/graphs/contributors">
  <img src="https://contributors-img.firebaseapp.com/image?repo=calindra/cartesi-rollups-hl-graphql" />
</a>

Made with [contributors-img](https://contributors-img.firebaseapp.com).
