package synchronizer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/repository"
	"github.com/jmoiron/sqlx"
)

type GraphileSynchronizer struct {
	Decoder                model.DecoderInterface
	SynchronizerRepository model.RepoSynchronizer
	GraphileFetcher        *GraphileFetcher
}

func (x GraphileSynchronizer) String() string {
	return "GraphileSynchronizer"
}

func (x GraphileSynchronizer) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}

	sleepInSeconds := 3

	lastFetch, err := x.SynchronizerRepository.GetLastFetched(ctx)

	if err != nil {
		return err
	}

	if lastFetch != nil {
		x.GraphileFetcher.CursorAfter = lastFetch.EndCursorAfter
		x.GraphileFetcher.CursorInputAfter = lastFetch.EndInputCursorAfter
		x.GraphileFetcher.CursorReportAfter = lastFetch.EndReportCursorAfter
	}

	for {
		voucherResp, err := x.GraphileFetcher.Fetch()

		if err != nil {
			slog.Warn(
				"Voucher fetcher error, we will try again",
				"error", err.Error(),
			)
		} else {
			err := x.handleWithDBTransaction(*voucherResp)
			if err != nil {
				slog.Error("Failed to handle graphile response.", "err", err)
			}
		}
		select {
		// Wait a little before doing another request
		case <-time.After(time.Duration(sleepInSeconds) * time.Second):
		case <-ctx.Done():
			slog.Debug("GraphileSynchronizer canceled:", "Error", ctx.Err().Error())
			return nil
		}

	}

}

func (x GraphileSynchronizer) handleWithDBTransaction(outputResp OutputResponse) error {
	const timeoutDuration = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	db := x.SynchronizerRepository.GetDB()
	ctxWithTx, err := repository.StartTransaction(ctx, db)

	if err != nil {
		slog.Error("Start db transaction fail.", "err", err)
		return err
	}

	tx, isTxEnable := repository.GetTransaction(ctxWithTx)

	if !isTxEnable {
		slog.Error("Recovery transaction fail.", "err", err)
		return err
	}

	err = x.handleGraphileResponse(ctxWithTx, outputResp)

	if err != nil {
		slog.Error("Failed to handle graphile response.", "err", err)
		x.callRollback(tx)
		return err
	}

	err = tx.Commit()
	if err != nil {
		slog.Error("Error attempting transaction commit.", "err", err)
		x.callRollback(tx)
		return err
	}

	return nil
}

func (x *GraphileSynchronizer) callRollback(tx *sqlx.Tx) {
	rollbackErr := tx.Rollback()
	if rollbackErr != nil {
		if rollbackErr != sql.ErrTxDone {
			slog.Error("Error attempting transaction rollback.", "err", rollbackErr)
		} else {
			slog.Warn("Transaction already completed, rollback not necessary.", "err", rollbackErr)
		}
	}
}

func (x GraphileSynchronizer) handleGraphileResponse(ctx context.Context, outputResp OutputResponse) error {
	// Handle response data
	voucherIds := []string{}
	initCursorAfter := x.GraphileFetcher.CursorAfter
	initInputCursorAfter := x.GraphileFetcher.CursorInputAfter
	initReportCursorAfter := x.GraphileFetcher.CursorReportAfter
	var endReportCursorAfter string
	var endInputCursorAfter string
	var endCursorAfter string

	for _, output := range outputResp.Data.Outputs.Edges {
		outputIndex := output.Node.Index
		inputIndex := output.Node.InputIndex

		processOutputData := model.ProcessOutputData{
			OutputIndex: uint64(outputIndex),
			InputIndex:  uint64(inputIndex),
			Payload:     output.Node.Blob,
			Destination: output.Node.Blob,
		}

		err := x.Decoder.HandleOutputV2(ctx, processOutputData)
		if err != nil {
			slog.Error("Failed to handle output: ", "err", err)
			return fmt.Errorf("error handling output: %w", err)
		}

		voucherIds = append(
			voucherIds,
			fmt.Sprintf("%d:%d", inputIndex, outputIndex),
		)
	}

	hasMoreOutputs := len(outputResp.Data.Outputs.PageInfo.EndCursor) > 0

	if hasMoreOutputs {
		initCursorAfter = x.GraphileFetcher.CursorAfter
		endCursorAfter = outputResp.Data.Outputs.PageInfo.EndCursor
	} else if len(initCursorAfter) > 0 {
		endCursorAfter = x.GraphileFetcher.CursorAfter
	}

	for _, input := range outputResp.Data.Inputs.Edges {

		slog.Debug("Add Input",
			"Index", input.Node.Index,
		)

		err := x.Decoder.HandleInput(ctx,
			input,
			model.CompletionStatusUnprocessed,
		)

		if err != nil {
			slog.Error("Failed to handle input:", "err", err)
			return fmt.Errorf("error handling input: %w", err)
		}
	}

	for _, report := range outputResp.Data.Reports.Edges {
		slog.Debug("Call HandleReport",
			"Index", report.Node.Index,
			"InputIndex", report.Node.InputIndex,
		)
		err := x.Decoder.HandleReport(
			ctx,
			report.Node.InputIndex,
			report.Node.Index,
			report.Node.Blob,
		)
		if err != nil {
			slog.Error("Failed to handle report:", "err", err)
			return fmt.Errorf("error handling report: %w", err)
		}
	}

	hasMoreReports := len(outputResp.Data.Reports.PageInfo.EndCursor) > 0
	if hasMoreReports {
		initReportCursorAfter = x.GraphileFetcher.CursorReportAfter
		endReportCursorAfter = outputResp.Data.Reports.PageInfo.EndCursor
	} else if len(initReportCursorAfter) > 0 {
		endReportCursorAfter = x.GraphileFetcher.CursorReportAfter
	}

	hasMoreInputs := len(outputResp.Data.Inputs.PageInfo.EndCursor) > 0

	if hasMoreInputs {
		initInputCursorAfter = x.GraphileFetcher.CursorInputAfter
		endInputCursorAfter = outputResp.Data.Inputs.PageInfo.EndCursor
	} else if len(initInputCursorAfter) > 0 {
		endInputCursorAfter = x.GraphileFetcher.CursorInputAfter
	}

	synchronizeFetch := &model.SynchronizerFetch{
		TimestampAfter:       uint64(time.Now().UnixMilli()),
		IniCursorAfter:       initCursorAfter,
		EndCursorAfter:       endCursorAfter,
		LogVouchersIds:       strings.Join(voucherIds, ";"),
		IniInputCursorAfter:  initInputCursorAfter,
		EndInputCursorAfter:  endInputCursorAfter,
		IniReportCursorAfter: initReportCursorAfter,
		EndReportCursorAfter: endReportCursorAfter,
	}

	if hasMoreInputs || hasMoreOutputs || hasMoreReports {
		_, err := x.SynchronizerRepository.Create(
			ctx, synchronizeFetch)
		if err != nil {
			slog.Error("Failed to create synchronize repository:", "err", err)
			return fmt.Errorf("error creating synchronize repository: %w", err)
		}
	}

	cursors := map[*string]string{
		&x.GraphileFetcher.CursorAfter:       endCursorAfter,
		&x.GraphileFetcher.CursorReportAfter: endReportCursorAfter,
		&x.GraphileFetcher.CursorInputAfter:  endInputCursorAfter,
	}

	for dest, cursor := range cursors {
		if len(cursor) != 0 {
			*dest = cursor
		}
	}

	return nil
}

func NewGraphileSynchronizer(
	decoder model.DecoderInterface,
	synchronizerRepository model.RepoSynchronizer,
	graphileFetcher *GraphileFetcher,
) *GraphileSynchronizer {
	return &GraphileSynchronizer{
		Decoder:                decoder,
		SynchronizerRepository: synchronizerRepository,
		GraphileFetcher:        graphileFetcher,
	}
}
