package reader

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.41

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/calindra/nonodo/internal/reader/graph"
	"github.com/calindra/nonodo/internal/reader/model"
)

// Voucher is the resolver for the voucher field.
func (r *inputResolver) Voucher(ctx context.Context, obj *model.Input, index int) (*model.Voucher, error) {
	return r.adapter.GetVoucher(index, obj.Index)
}

// Notice is the resolver for the notice field.
func (r *inputResolver) Notice(ctx context.Context, obj *model.Input, index int) (*model.Notice, error) {
	return r.adapter.GetNotice(index, obj.Index)
}

// Report is the resolver for the report field.
func (r *inputResolver) Report(ctx context.Context, obj *model.Input, index int) (*model.Report, error) {
	return r.adapter.GetReport(index, obj.Index)
}

// Vouchers is the resolver for the vouchers field.
func (r *inputResolver) Vouchers(ctx context.Context, obj *model.Input, first *int, last *int, after *string, before *string) (*model.Connection[*model.Voucher], error) {
	return r.adapter.GetVouchers(first, last, after, before, &obj.Index)
}

// Notices is the resolver for the notices field.
func (r *inputResolver) Notices(ctx context.Context, obj *model.Input, first *int, last *int, after *string, before *string) (*model.Connection[*model.Notice], error) {
	return r.adapter.GetNotices(first, last, after, before, &obj.Index)
}

// Reports is the resolver for the reports field.
func (r *inputResolver) Reports(ctx context.Context, obj *model.Input, first *int, last *int, after *string, before *string) (*model.Connection[*model.Report], error) {
	return r.adapter.GetReports(ctx, first, last, after, before, &obj.Index)
}

// Input is the resolver for the input field.
func (r *noticeResolver) Input(ctx context.Context, obj *model.Notice) (*model.Input, error) {
	return r.adapter.GetInput(strconv.Itoa(obj.InputIndex))
}

// Proof is the resolver for the proof field.
func (r *noticeResolver) Proof(ctx context.Context, obj *model.Notice) (*model.Proof, error) {
	return r.adapter.GetProof(ctx, obj.InputIndex, obj.Index)
}

// Input is the resolver for the input field.
func (r *queryResolver) Input(ctx context.Context, id string) (*model.Input, error) {
	slog.Debug("queryResolver.Input", "id", id)
	return r.adapter.GetInput(id)
}

// Inputs is the resolver for the inputs field.
func (r *queryResolver) Inputs(ctx context.Context, first *int, last *int, after *string, before *string, where *model.InputFilter) (*model.Connection[*model.Input], error) {
	return r.adapter.GetInputs(ctx, first, last, after, before, where)
}

// Vouchers is the resolver for the vouchers field.
func (r *queryResolver) Vouchers(ctx context.Context, first *int, last *int, after *string, before *string, filter []*model.ConvenientFilter) (*model.Connection[*model.Voucher], error) {
	convenienceFilter, err := model.ConvertToConvenienceFilter(filter)
	if err != nil {
		return nil, err
	}
	vouchers, err := r.convenienceService.FindAllVouchers(ctx, first, last, after, before, convenienceFilter)
	if err != nil {
		return nil, err
	}
	return model.ConvertToVoucherConnectionV1(vouchers.Rows, int(vouchers.Offset), int(vouchers.Total))
}

// Notices is the resolver for the notices field.
func (r *queryResolver) Notices(ctx context.Context, first *int, last *int, after *string, before *string) (*model.Connection[*model.Notice], error) {
	return r.adapter.GetNotices(first, last, after, before, nil)
}

// Reports is the resolver for the reports field.
func (r *queryResolver) Reports(ctx context.Context, first *int, last *int, after *string, before *string) (*model.Connection[*model.Report], error) {
	return r.adapter.GetReports(ctx, first, last, after, before, nil)
}

// Input is the resolver for the input field.
func (r *reportResolver) Input(ctx context.Context, obj *model.Report) (*model.Input, error) {
	return r.adapter.GetInput(strconv.Itoa(obj.InputIndex))
}

// Input is the resolver for the input field.
func (r *voucherResolver) Input(ctx context.Context, obj *model.Voucher) (*model.Input, error) {
	return r.adapter.GetInput(strconv.Itoa(obj.InputIndex))
}

// Proof is the resolver for the proof field.
func (r *voucherResolver) Proof(ctx context.Context, obj *model.Voucher) (*model.Proof, error) {
	return r.adapter.GetProof(ctx, obj.InputIndex, obj.Index)
}

// Input returns graph.InputResolver implementation.
func (r *Resolver) Input() graph.InputResolver { return &inputResolver{r} }

// Notice returns graph.NoticeResolver implementation.
func (r *Resolver) Notice() graph.NoticeResolver { return &noticeResolver{r} }

// Query returns graph.QueryResolver implementation.
func (r *Resolver) Query() graph.QueryResolver { return &queryResolver{r} }

// Report returns graph.ReportResolver implementation.
func (r *Resolver) Report() graph.ReportResolver { return &reportResolver{r} }

// Voucher returns graph.VoucherResolver implementation.
func (r *Resolver) Voucher() graph.VoucherResolver { return &voucherResolver{r} }

type inputResolver struct{ *Resolver }
type noticeResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type reportResolver struct{ *Resolver }
type voucherResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *queryResolver) Voucher(ctx context.Context, voucherIndex int, inputIndex int) (*model.Voucher, error) {
	slog.Debug("queryResolver.Voucher", "voucherIndex", voucherIndex, "inputIndex", inputIndex)
	return r.adapter.GetVoucher(voucherIndex, inputIndex)
}
func (r *queryResolver) Notice(ctx context.Context, noticeIndex int, inputIndex int) (*model.Notice, error) {
	slog.Debug("queryResolver.Notice", "noticeIndex", noticeIndex, "inputIndex", inputIndex)
	return r.adapter.GetNotice(noticeIndex, inputIndex)
}
func (r *queryResolver) Report(ctx context.Context, reportIndex int, inputIndex int) (*model.Report, error) {
	return r.adapter.GetReport(reportIndex, inputIndex)
}
