package reader

import (
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/services"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/reader/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	model              *model.ModelWrapper
	convenienceService *services.ConvenienceService
	adapter            Adapter
}
