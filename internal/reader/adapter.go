package reader

import (
	"context"

	graphql "github.com/calindra/cartesi-rollups-hl-graphql/internal/reader/model"
)

type Adapter interface {
	GetReport(reportIndex int, inputIndex int) (*graphql.Report, error)

	GetReports(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetInputs(
		ctx context.Context,
		first *int, last *int, after *string, before *string, where *graphql.InputFilter,
	) (*graphql.InputConnection, error)

	GetInput(index int) (*graphql.Input, error)

	GetNotice(noticeIndex int, inputIndex int) (*graphql.Notice, error)

	GetNotices(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.NoticeConnection, error)

	GetVoucher(voucherIndex int, inputIndex int) (*graphql.Voucher, error)

	GetVouchers(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.VoucherConnection, error)

	GetProof(ctx context.Context, inputIndex, outputIndex int) (*graphql.Proof, error)
}
