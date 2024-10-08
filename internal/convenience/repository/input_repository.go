package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/calindra/cartesi-rollups-hl-graphql/internal/commons"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const INDEX_FIELD = "Index"

type InputRepository struct {
	Db sqlx.DB
}

type inputRow struct {
	Index          int    `db:"input_index"`
	Status         int    `db:"status"`
	MsgSender      string `db:"msg_sender"`
	Payload        string `db:"payload"`
	BlockNumber    int    `db:"block_number"`
	BlockTimestamp int    `db:"block_timestamp"`
	PrevRandao     string `db:"prev_randao"`
	Exception      string `db:"exception"`
	AppContract    string `db:"app_contract"`
}

func (r *InputRepository) CreateTables() error {
	autoIncrement := "INTEGER"

	if r.Db.DriverName() == "postgres" {
		autoIncrement = "SERIAL"
	}

	schema := `CREATE TABLE IF NOT EXISTS convenience_inputs (
		id 				%s NOT NULL PRIMARY KEY,
		input_index		integer,
		app_contract    text,
		status	 		text,
		msg_sender	 	text,
		payload			text,
		block_number	integer,
		block_timestamp	integer,
		prev_randao		text,
		exception		text);
	CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_inputs(input_index);
	CREATE INDEX IF NOT EXISTS idx_status ON convenience_inputs(status);`
	schema = fmt.Sprintf(schema, autoIncrement)
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Inputs table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *InputRepository) Create(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	exist, err := r.FindByIndex(ctx, input.Index)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		return exist, nil
	}
	return r.rawCreate(ctx, input)
}

func (r *InputRepository) RawCreate(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	return r.rawCreate(ctx, input)
}

func (r *InputRepository) rawCreate(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	insertSql := `INSERT INTO convenience_inputs (
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		app_contract
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9
	);`

	exec := DBExecutor{&r.Db}
	_, err := exec.ExecContext(
		ctx,
		insertSql,
		input.Index,
		input.Status,
		input.MsgSender.Hex(),
		common.Bytes2Hex(input.Payload),
		input.BlockNumber,
		input.BlockTimestamp.UnixMilli(),
		input.PrevRandao,
		common.Bytes2Hex(input.Exception),
		input.AppContract.Hex(),
	)

	if err != nil {
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) Update(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	sql := `UPDATE convenience_inputs
		SET status = $1, exception = $2
		WHERE input_index = $3`

	exec := DBExecutor{&r.Db}
	_, err := exec.ExecContext(
		ctx,
		sql,
		input.Status,
		common.Bytes2Hex(input.Exception),
		input.Index,
	)
	if err != nil {
		slog.Error("Error updating voucher", "Error", err)
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) FindByStatusNeDesc(ctx context.Context, status model.CompletionStatus) (*model.AdvanceInput, error) {
	sql := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception,
		app_contract FROM convenience_inputs WHERE status <> $1
		ORDER BY input_index DESC`
	res, err := r.Db.QueryxContext(
		ctx,
		sql,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindByStatus(ctx context.Context, status model.CompletionStatus) (*model.AdvanceInput, error) {
	sql := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		app_contract FROM convenience_inputs WHERE status = $1
		ORDER BY input_index ASC`
	res, err := r.Db.QueryxContext(
		ctx,
		sql,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindByIndex(ctx context.Context, index int) (*model.AdvanceInput, error) {
	sql := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		app_contract FROM convenience_inputs WHERE input_index = $1`
	res, err := r.Db.QueryxContext(
		ctx,
		sql,
		index,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindInputByAppContractAndIndex(ctx context.Context, index int, appContract common.Address) (*model.AdvanceInput, error) {
	sql := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		app_contract FROM convenience_inputs WHERE input_index = $1 AND app_contract = $2`

	res, err := r.Db.QueryxContext(
		ctx,
		sql,
		index,
		appContract.Hex(),
	)

	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (c *InputRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_inputs `
	where, args, _, err := transformToInputQuery(filter)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, args...)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *InputRepository) FindAll(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.AdvanceInput], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		app_contract FROM convenience_inputs `
	where, args, argsCount, err := transformToInputQuery(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC `

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
		slog.Error("Find all error", "error", err)
		return nil, err
	}
	defer stmt.Close()
	var rows []inputRow
	erro := stmt.SelectContext(ctx, &rows, args...)
	if erro != nil {
		slog.Error("Find all error", "error", erro)
		return nil, erro
	}

	inputs := make([]model.AdvanceInput, len(rows))

	for i, row := range rows {
		inputs[i] = parseRowInput(row)
	}

	pageResult := &commons.PageResult[model.AdvanceInput]{
		Rows:   inputs,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func transformToInputQuery(
	filter []*model.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += WHERE
	}
	args := []interface{}{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == INDEX_FIELD {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else if filter.Gt != nil {
				where = append(where, fmt.Sprintf("input_index > $%d ", count))
				args = append(args, *filter.Gt)
				count += 1
			} else if filter.Lt != nil {
				where = append(where, fmt.Sprintf("input_index < $%d ", count))
				args = append(args, *filter.Lt)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == "Status" {
			if filter.Ne != nil {
				where = append(where, fmt.Sprintf("status <> $%d ", count))
				args = append(args, *filter.Ne)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == "MsgSender" {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("msg_sender = $%d ", count))
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

func parseRowInput(row inputRow) model.AdvanceInput {
	return model.AdvanceInput{
		Index:          row.Index,
		Status:         model.CompletionStatus(row.Status),
		MsgSender:      common.HexToAddress(row.MsgSender),
		Payload:        common.Hex2Bytes(row.Payload),
		BlockNumber:    uint64(row.BlockNumber),
		BlockTimestamp: time.UnixMilli(int64(row.BlockTimestamp)),
		PrevRandao:     row.PrevRandao,
		Exception:      common.Hex2Bytes(row.Exception),
		AppContract:    common.HexToAddress(row.AppContract),
	}
}

func parseInput(res *sqlx.Rows) (*model.AdvanceInput, error) {
	var (
		input          model.AdvanceInput
		msgSender      string
		payload        string
		blockTimestamp int64
		prevRandao     string
		exception      string
		appContract    string
	)
	err := res.Scan(
		&input.Index,
		&input.Status,
		&msgSender,
		&payload,
		&input.BlockNumber,
		&blockTimestamp,
		&prevRandao,
		&exception,
		&appContract,
	)
	if err != nil {
		return nil, err
	}
	input.Payload = common.Hex2Bytes(payload)
	input.MsgSender = common.HexToAddress(msgSender)
	input.BlockTimestamp = time.UnixMilli(blockTimestamp)
	input.PrevRandao = prevRandao
	input.Exception = common.Hex2Bytes(exception)
	input.AppContract = common.HexToAddress(appContract)
	return &input, nil
}
