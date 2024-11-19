// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	cModel "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/model"
	cRepos "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/repository"

	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/commons"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience"
	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type ModelSuite struct {
	suite.Suite
	m                  *NonodoModel
	n                  int
	payloads           []string
	senders            []common.Address
	blockNumbers       []uint64
	timestamps         []time.Time
	reportRepository   *cRepos.ReportRepository
	inputRepository    *cRepos.InputRepository
	voucherRepository  *cRepos.VoucherRepository
	noticeRepository   *cRepos.NoticeRepository
	tempDir            string
	convenienceService *services.ConvenienceService
}

func (s *ModelSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "")
	s.tempDir = tempDir
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	sqliteFileName = path.Join(tempDir, sqliteFileName)
	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	container := convenience.NewContainer(*db, false)
	decoder := container.GetOutputDecoder()
	s.reportRepository = container.GetReportRepository()
	s.inputRepository = container.GetInputRepository()
	s.voucherRepository = container.GetVoucherRepository()
	s.noticeRepository = container.GetNoticeRepository()

	s.m = NewNonodoModel(
		decoder,
		s.reportRepository,
		s.inputRepository,
		s.voucherRepository,
		s.noticeRepository,
	)
	s.convenienceService = container.GetConvenienceService()
	s.n = 3
	s.payloads = make([]string, s.n)
	s.senders = make([]common.Address, s.n)
	s.blockNumbers = make([]uint64, s.n)
	s.timestamps = make([]time.Time, s.n)
	now := time.Now()
	for i := 0; i < s.n; i++ {
		for addrI := 0; addrI < common.AddressLength; addrI++ {
			s.senders[i][addrI] = 0xf0 + byte(i)
		}
		s.payloads[i] = common.Bytes2Hex([]byte{0xf0 + byte(i)})
		s.blockNumbers[i] = uint64(i)
		s.timestamps[i] = now.Add(time.Second * time.Duration(i))
	}
}

func TestModelSuite(t *testing.T) {
	suite.Run(t, new(ModelSuite))
}

//
// GetVouchers
//

func (s *ModelSuite) TestItGetsNoVouchers() {
	vouchers := s.getAllVouchers(0, 100, nil)
	s.Empty(vouchers)
}

//
// GetNotices
//

func (s *ModelSuite) TestItGetsNoNotices() {
	ctx := context.Background()
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(notices.Rows)
}

//
// GetReports
//

func (s *ModelSuite) TestItGetsNoReports() {
	ctx := context.Background()
	reports, err := s.reportRepository.FindAll(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func (s *ModelSuite) getAllVouchers(
	offset int, limit int, inputIndex *int,
) []cModel.ConvenienceVoucher {
	ctx := context.Background()
	filters := []*cModel.ConvenienceFilter{}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	if offset != 0 {
		afterOffset := commons.EncodeCursor(offset - 1)
		vouchers, err := s.convenienceService.
			FindAllVouchers(ctx, &limit, nil, &afterOffset, nil, filters)
		s.NoError(err)
		return vouchers.Rows
	} else {
		vouchers, err := s.convenienceService.
			FindAllVouchers(ctx, &limit, nil, nil, nil, filters)
		s.NoError(err)
		return vouchers.Rows
	}
}
