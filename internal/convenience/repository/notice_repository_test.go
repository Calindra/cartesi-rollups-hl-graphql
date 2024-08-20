package repository

import (
	"context"
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

type NoticeRepositorySuite struct {
	suite.Suite
	repository *NoticeRepository
}

func (s *NoticeRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &NoticeRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	s.NoError(err)
}

func TestNoticeRepositorySuite(t *testing.T) {
	suite.Run(t, new(NoticeRepositorySuite))
}

func (s *NoticeRepositorySuite) TestCreateNotice() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *NoticeRepositorySuite) TestFindByInputAndOutputIndex() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		AppContract: common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	})
	s.NoError(err)
	notice, err := s.repository.FindByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	s.Equal("0x0011", notice.Payload)
	s.Equal(1, int(notice.InputIndex))
	s.Equal(2, int(notice.OutputIndex))
	s.Equal("0x70997970C51812dc3A010C7d01b50e0d17dc79C8", notice.AppContract.Hex())
}

func (s *NoticeRepositorySuite) TestCountNotices() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	_, err = s.repository.Create(ctx, &model.ConvenienceNotice{
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 0,
	})
	s.NoError(err)
	total, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(2, int(total))

	filters := []*model.ConvenienceFilter{}
	{
		field := "InputIndex"
		value := "2"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	total, err = s.repository.Count(ctx, filters)
	s.NoError(err)
	s.Equal(1, int(total))
}

func (s *NoticeRepositorySuite) TestNoticePagination() {
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
			Payload:     "0x0011",
			InputIndex:  uint64(i),
			OutputIndex: 0,
		})
		s.NoError(err)
	}

	total, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(30, int(total))

	filters := []*model.ConvenienceFilter{}
	first := 10
	notices, err := s.repository.FindAllNotices(ctx, &first, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(0, int(notices.Rows[0].InputIndex))
	s.Equal(9, int(notices.Rows[len(notices.Rows)-1].InputIndex))

	after := commons.EncodeCursor(10)
	notices, err = s.repository.FindAllNotices(ctx, &first, nil, &after, nil, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(11, int(notices.Rows[0].InputIndex))
	s.Equal(20, int(notices.Rows[len(notices.Rows)-1].InputIndex))

	last := 10
	notices, err = s.repository.FindAllNotices(ctx, nil, &last, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(20, int(notices.Rows[0].InputIndex))
	s.Equal(29, int(notices.Rows[len(notices.Rows)-1].InputIndex))

	before := commons.EncodeCursor(20)
	notices, err = s.repository.FindAllNotices(ctx, nil, &last, nil, &before, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(10, int(notices.Rows[0].InputIndex))
	s.Equal(19, int(notices.Rows[len(notices.Rows)-1].InputIndex))
}

func (s *NoticeRepositorySuite) TestFindReportByAppContractAndIndex() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 1,
		AppContract: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})
	s.NoError(err)

	_, err = s.repository.Create(ctx, &model.ConvenienceNotice{
		Payload:     "0xFF22",
		InputIndex:  2,
		OutputIndex: 3,
		AppContract: common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"),
	})
	s.NoError(err)

	report, err := s.repository.FindNoticeByAppContractAndIndex(ctx, 2, common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"))
	s.NoError(err)

	s.Equal(common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"), report.AppContract)
	s.Equal("0xFF22", report.Payload)
	s.Equal(uint64(2), report.InputIndex)
	s.Equal(uint64(3), report.OutputIndex)
}
