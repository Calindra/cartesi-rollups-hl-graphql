package repository

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/calindra/cartesi-rollups-hl-graphql/internal/commons"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

type VoucherRepositorySuite struct {
	suite.Suite
	repository *VoucherRepository
}

func (s *VoucherRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &VoucherRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	s.NoError(err)
}

func TestConvenienceRepositorySuite(t *testing.T) {
	suite.Run(t, new(VoucherRepositorySuite))
}

func (s *VoucherRepositorySuite) TestCreateVoucher() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *VoucherRepositorySuite) TestFindVoucher() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
		AppContract: common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	})
	s.NoError(err)
	voucher, err := s.repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(1, int(voucher.InputIndex))
	s.Equal(2, int(voucher.OutputIndex))
	s.Equal(false, voucher.Executed)
	s.Equal("0x70997970C51812dc3A010C7d01b50e0d17dc79C8", voucher.AppContract.Hex())
}

func (s *VoucherRepositorySuite) TestFindVoucherExecuted() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	s.NoError(err)
	voucher, err := s.repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(1, int(voucher.InputIndex))
	s.Equal(2, int(voucher.OutputIndex))
	s.Equal(true, voucher.Executed)
}

func (s *VoucherRepositorySuite) TestCountVoucher() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	s.NoError(err)
	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 0,
		Executed:    false,
	})
	s.NoError(err)
	total, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(2, int(total))

	filters := []*model.ConvenienceFilter{}
	{
		field := "Executed"
		value := "false"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	total, err = s.repository.Count(ctx, filters)
	s.NoError(err)
	s.Equal(1, int(total))
}

func (s *VoucherRepositorySuite) TestPagination() {
	destination := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
			Destination: destination,
			Payload:     "0x0011",
			InputIndex:  uint64(i),
			OutputIndex: 0,
			Executed:    false,
		})
		s.NoError(err)
	}

	total, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(30, int(total))

	filters := []*model.ConvenienceFilter{}
	{
		field := "Executed"
		value := "false"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	first := 10
	vouchers, err := s.repository.FindAllVouchers(ctx, &first, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(0, int(vouchers.Rows[0].InputIndex))
	s.Equal(9, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))

	after := commons.EncodeCursor(10)
	vouchers, err = s.repository.FindAllVouchers(ctx, &first, nil, &after, nil, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(11, int(vouchers.Rows[0].InputIndex))
	s.Equal(20, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))

	last := 10
	vouchers, err = s.repository.FindAllVouchers(ctx, nil, &last, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(20, int(vouchers.Rows[0].InputIndex))
	s.Equal(29, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))

	before := commons.EncodeCursor(20)
	vouchers, err = s.repository.FindAllVouchers(ctx, nil, &last, nil, &before, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(10, int(vouchers.Rows[0].InputIndex))
	s.Equal(19, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))
}

func (s *VoucherRepositorySuite) TestWrongAddress() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	s.NoError(err)
	filters := []*model.ConvenienceFilter{}
	{
		field := "Destination"
		value := "0xError"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	_, err = s.repository.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	if err == nil {
		s.Fail("where is the error?")
	}
	s.Equal("wrong address value", err.Error())
}

func (s *VoucherRepositorySuite) TestFindVoucherByAppContractAndIndex() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 4,
		AppContract: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})
	s.NoError(err)

	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Payload:     "0xFF22",
		InputIndex:  2,
		OutputIndex: 3,
		AppContract: common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"),
	})
	s.NoError(err)

	voucher, err := s.repository.FindVoucherByAppContractAndIndex(ctx, 1, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"))
	s.NoError(err)

	s.Equal(common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), voucher.AppContract)
	s.Equal("0x0011", voucher.Payload)
	s.Equal(uint64(1), voucher.InputIndex)
	s.Equal(uint64(4), voucher.OutputIndex)
}
