package rollup

import (
	"net/http"

	DA "github.com/calindra/cartesi-rollups-hl-graphql/internal/dataavailability"
	"github.com/labstack/echo/v4"
)

func (r *RollupAPI) Fetcher(ctx echo.Context, request GioJSONRequestBody) (*GioResponseRollup, *DA.HttpCustomError) {
	var (
		syscoin  uint16 = 5700
		celestia uint16 = 714
		avail    uint16 = 9944
		its_ok   uint16 = 42
	)

	switch request.Domain {
	case syscoin:
		syscoinFetcher := DA.NewSyscoinClient()
		data, err := syscoinFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case celestia:
		celestiaFetcher := DA.NewCelestiaClient()
		data, err := celestiaFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case avail:
		availFetcher := DA.NewAvailFetcher()
		data, err := availFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	default:
		unsupported := "Unsupported domain"
		return nil, DA.NewHttpCustomError(http.StatusBadRequest, &unsupported)
	}
}
