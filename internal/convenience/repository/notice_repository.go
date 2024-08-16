package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/cartesi-rollups-hl-graphql/internal/commons"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type NoticeRepository struct {
	Db sqlx.DB
}

type noticeRow struct {
	Payload     string `db:"payload"`
	Index       int    `db:"input_index"`
	Output      int    `db:"output_index"`
	AppContract string `db:"app_contract"`
}

func (c *NoticeRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_notices (
		payload 		text,
		input_index		integer,
		output_index	integer,
		app_contract    text,
		PRIMARY KEY (input_index, output_index));`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *NoticeRepository) Create(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	insertSql := `INSERT INTO convenience_notices (
		payload,
		input_index,
		output_index,
		app_contract) VALUES ($1, $2, $3, $4)`

	exec := DBExecutor{&c.Db}
	_, err := exec.ExecContext(ctx,
		insertSql,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
		data.AppContract.Hex(),
	)
	if err != nil {
		slog.Error("Error creating convenience_notice", "Error", err)
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) Update(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	sqlUpdate := `UPDATE convenience_notices SET 
		payload = $1
		WHERE input_index = $2 and output_index = $3`
	exec := DBExecutor{&c.Db}
	_, err := exec.ExecContext(
		ctx,
		sqlUpdate,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
	)
	if err != nil {
		slog.Error("Error updating convenience_notice", "Error", err)
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_notices `
	where, args, _, err := transformToNoticeQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (c *NoticeRepository) FindAllNotices(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceNotice], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM convenience_notices `
	where, args, argsCount, err := transformToNoticeQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC, output_index ASC `
	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf("LIMIT $%d ", argsCount)
	args = append(args, limit)
	argsCount = argsCount + 1
	query += fmt.Sprintf("OFFSET $%d ", argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var rows []noticeRow
	err = stmt.SelectContext(ctx, &rows, args...)
	if err != nil {
		return nil, err
	}

	notices := make([]model.ConvenienceNotice, len(rows))

	for i, row := range rows {
		notices[i] = parseRowNotice(row)
	}
	pageResult := &commons.PageResult[model.ConvenienceNotice]{
		Rows:   notices,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func (c *NoticeRepository) FindByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	query := `SELECT * FROM convenience_notices WHERE input_index = $1 and output_index = $2 LIMIT 1`
	res, err := c.Db.QueryxContext(
		ctx,
		query,
		inputIndex,
		outputIndex,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if res.Next() {
		notice, err := parseNotice(res)
		if err != nil {
			return nil, err
		}
		slog.Debug("RETORNOU NOTICE", "notice", notice)
		return notice, nil
	}

	return nil, nil
}

func transformToNoticeQuery(
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
		if *filter.Field == model.INPUT_INDEX {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("input_index = $%d ", count),
				)
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
	slog.Debug("Query", "query", query, "args", args)
	return query, args, count, nil
}

func parseNotice(res *sqlx.Rows) (*model.ConvenienceNotice, error) {
	var (
		notice model.ConvenienceNotice
		// payload     string
		appContract string
	)
	err := res.Scan(
		&notice.Payload,
		&notice.InputIndex,
		&notice.OutputIndex,
		&appContract,
	)
	if err != nil {
		slog.Error("ERROR PARSENOTICE", "error", err)
		return nil, err
	}
	slog.Error("NÃO APRESENTOU ERRO")
	notice.AppContract = common.HexToAddress(appContract)

	return &notice, nil
}

func parseRowNotice(row noticeRow) model.ConvenienceNotice {
	return model.ConvenienceNotice{
		Payload:     row.Payload,
		InputIndex:  uint64(row.Index),
		OutputIndex: uint64(row.Output),
		AppContract: common.HexToAddress(row.AppContract),
	}
}
