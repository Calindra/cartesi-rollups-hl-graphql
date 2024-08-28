// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package is responsible for serving the GraphQL reader API.
package reader

//go:generate go run github.com/99designs/gqlgen generate

import (
	"context"
	"log/slog"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	cModel "github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/services"
	nonodomodel "github.com/calindra/cartesi-rollups-hl-graphql/internal/model"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/reader/graph"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/reader/model"
	"github.com/labstack/echo/v4"
)

// Register the GraphQL reader API to echo.
func Register(
	e *echo.Echo,
	nonodomodel *nonodomodel.NonodoModel,
	convenienceService *services.ConvenienceService,
	adapter Adapter,
) {
	resolver := Resolver{
		model.NewModelWrapper(nonodomodel),
		convenienceService,
		adapter,
	}
	config := graph.Config{Resolvers: &resolver}
	schema := graph.NewExecutableSchema(config)
	graphqlHandler := handler.NewDefaultServer(schema)
	playgroundHandler := playground.Handler("GraphQL", "/graphql")
	e.POST("/graphql", func(c echo.Context) error {
		graphqlHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	e.POST("/:appContract/graphql", func(c echo.Context) error {
		appContract := c.Param("appContract")
		slog.Debug("path parameter received: ", "app_contract", appContract)
		ctx := context.WithValue(c.Request().Context(), cModel.AppContractKey, appContract)
		c.SetRequest(c.Request().WithContext(ctx))
		graphqlHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	e.GET("/graphql", func(c echo.Context) error {
		playgroundHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
