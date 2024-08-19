package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/cartesi-rollups-hl-graphql/internal/commons"
	cModel "github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const INPUT_INDEX = "InputIndex"

type ReportRepository struct {
	Db *sqlx.DB
}

type reportRow struct {
	Index       int    `db:"output_index"`
	InputIndex  int    `db:"input_index"`
	Payload     string `db:"payload"`
	AppContract string `db:"app_contract"`
}

func (r *ReportRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_reports (
		output_index	integer,
		payload 		text,
		input_index 	integer,
		app_contract    text,
		PRIMARY KEY (input_index, output_index));`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Reports table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *ReportRepository) Create(ctx context.Context, report cModel.Report) (cModel.Report, error) {
	insertSql := `INSERT INTO convenience_reports (
		output_index,
		payload,
		input_index,
		app_contract) VALUES ($1, $2, $3, $4)`

	exec := DBExecutor{r.Db}
	_, err := exec.ExecContext(
		ctx,
		insertSql,
		report.Index,
		common.Bytes2Hex(report.Payload),
		report.InputIndex,
		report.AppContract.Hex(),
	)

	if err != nil {
		slog.Error("database error", "err", err)
		return cModel.Report{}, err
	}
	slog.Debug("Report created",
		"outputIndex", report.Index,
		"inputIndex", report.InputIndex,
	)
	return report, nil
}

func (r *ReportRepository) Update(ctx context.Context, report cModel.Report) (*cModel.Report, error) {
	sql := `UPDATE convenience_reports
		SET payload = $1
		WHERE input_index = $2 and output_index = $3 `

	exec := DBExecutor{r.Db}
	_, err := exec.ExecContext(
		ctx,
		sql,
		common.Bytes2Hex(report.Payload),
		report.InputIndex,
		report.Index,
	)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepository) FindByInputAndOutputIndex(
	ctx context.Context,
	inputIndex uint64,
	outputIndex uint64,
) (*cModel.Report, error) {
	rows, err := r.Db.QueryxContext(ctx, `
		SELECT payload FROM convenience_reports
		WHERE input_index = $1 AND output_index = $2
		LIMIT 1`,
		inputIndex, outputIndex,
	)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: int(inputIndex),
			Index:      int(outputIndex),
			Payload:    common.Hex2Bytes(payload),
		}
		return report, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *ReportRepository) FindInputByAppContractAndIndex(ctx context.Context, index int, appContract common.Address) (*cModel.Report, error) {

	query := `SELECT 
		input_index, 
		output_index, 
		payload, 
		app_contract FROM convenience_reports WHERE input_index = $1 AND app_contract = $2`

	res, err := r.Db.QueryxContext(
		ctx,
		query,
		uint64(index),
		appContract.Hex(),
	)

	if err != nil {
		return nil, err
	}
	defer res.Close()

	if res.Next() {
		report, err := parseReport(res)
		if err != nil {
			return nil, err
		}
		return report, nil
	}
	return nil, nil
}

func (c *ReportRepository) Count(
	ctx context.Context,
	filter []*cModel.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_reports `
	where, args, _, err := transformToReportQuery(filter)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.Get(&count, args...)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *ReportRepository) FindAllByInputIndex(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*commons.PageResult[cModel.Report], error) {
	filter := []*cModel.ConvenienceFilter{}
	if inputIndex != nil {
		field := INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filter = append(filter, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	return c.FindAll(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ReportRepository) FindAll(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*cModel.ConvenienceFilter,
) (*commons.PageResult[cModel.Report], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}

	query := `SELECT input_index, output_index, payload, app_contract FROM convenience_reports `
	where, args, argsCount, err := transformToReportQuery(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC, output_index ASC `

	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf(`LIMIT $%d `, argsCount)
	args = append(args, limit)
	argsCount += 1
	query += fmt.Sprintf(`OFFSET $%d `, argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var rows []reportRow
	err = stmt.SelectContext(ctx, &rows, args...)

	if err != nil {
		slog.Error("Find all error", "error", err)
		return nil, err
	}
	reports := make([]cModel.Report, len(rows))

	for i, row := range rows {
		reports[i] = parseReportRow(row)
	}

	pageResult := &commons.PageResult[cModel.Report]{
		Rows:   reports,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func transformToReportQuery(
	filter []*cModel.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += WHERE
	}
	args := []interface{}{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == "OutputIndex" {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("output_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == INPUT_INDEX {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else {
			return "", nil, 0, fmt.Errorf("unexpected field %s", *filter.Field)
		}
	}
	query += strings.Join(where, " and ")
	return query, args, count, nil
}

func parseReportRow(row reportRow) cModel.Report {
	return cModel.Report{
		Index:       row.Index,
		InputIndex:  row.InputIndex,
		Payload:     common.Hex2Bytes(row.Payload),
		AppContract: common.HexToAddress(row.AppContract),
	}
}

func parseReport(res *sqlx.Rows) (*cModel.Report, error) {
	var (
		report      cModel.Report
		payload     string
		appContract string
	)
	err := res.Scan(
		&report.InputIndex,
		&report.Index,
		&payload,
		&appContract,
	)
	if err != nil {
		return nil, err
	}

	report.Payload = common.Hex2Bytes(payload)
	report.AppContract = common.HexToAddress(appContract)
	return &report, nil
}
