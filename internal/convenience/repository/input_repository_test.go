package repository

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	convenience "github.com/calindra/cartesi-rollups-hl-graphql/internal/convenience/model"

	"github.com/calindra/cartesi-rollups-hl-graphql/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type InputRepositorySuite struct {
	suite.Suite
	inputRepository *InputRepository
	tempDir         string
}

func (s *InputRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	tempDir, err := os.MkdirTemp("", "")
	s.tempDir = tempDir
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("input%d.sqlite3", rand.Intn(1000))
	sqliteFileName = path.Join(tempDir, sqliteFileName)
	// db := sqlx.MustConnect("sqlite3", ":memory:")
	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.inputRepository = &InputRepository{
		Db: *db,
	}
	err = s.inputRepository.CreateTables()
	s.NoError(err)
}

func TestInputRepositorySuite(t *testing.T) {
	// t.Parallel()
	suite.Run(t, new(InputRepositorySuite))
}

func (s *InputRepositorySuite) TestCreateTables() {
	defer s.teardown()
	err := s.inputRepository.CreateTables()
	s.NoError(err)
}

func (s *InputRepositorySuite) TestCreateInput() {
	defer s.teardown()
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          0,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
}

func (s *InputRepositorySuite) TestFixCreateInputDuplicated() {
	defer s.teardown()
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          0,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
	input, err = s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          0,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
	count, err := s.inputRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(uint64(1), count)
}

func (s *InputRepositorySuite) TestCreateAndFindInputByIndex() {
	defer s.teardown()
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          123,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		Payload:        common.Hex2Bytes("1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(123, input.Index)

	input2, err := s.inputRepository.FindByIndex(ctx, 123)
	s.NoError(err)
	s.Equal(123, input.Index)
	s.Equal(input.Status, input2.Status)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", input.MsgSender.Hex())
	s.Equal("1122", common.Bytes2Hex(input.Payload))
	s.Equal(1, int(input2.BlockNumber))
	s.Equal(input.BlockTimestamp.UnixMilli(), input2.BlockTimestamp.UnixMilli())
}

func (s *InputRepositorySuite) TestCreateInputAndUpdateStatus() {
	defer s.teardown()
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	})
	s.NoError(err)
	s.Equal(2222, input.Index)

	input.Status = convenience.CompletionStatusAccepted
	_, err = s.inputRepository.Update(ctx, *input)
	s.NoError(err)

	input2, err := s.inputRepository.FindByIndex(ctx, 2222)
	s.NoError(err)
	s.Equal(convenience.CompletionStatusAccepted, input2.Status)
	s.Equal("0x70997970C51812dc3A010C7d01b50e0d17dc79C8", input2.AppContract.Hex())
}

func (s *InputRepositorySuite) TestCreateInputFindByStatus() {
	defer s.teardown()
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		PrevRandao:     "0xdeadbeef",
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})
	s.NoError(err)
	s.Equal(2222, input.Index)

	input2, err := s.inputRepository.FindByStatus(ctx, convenience.CompletionStatusUnprocessed)
	s.NoError(err)
	s.Equal(2222, input2.Index)

	input.Status = convenience.CompletionStatusAccepted
	_, err = s.inputRepository.Update(ctx, *input)
	s.NoError(err)

	input2, err = s.inputRepository.FindByStatus(ctx, convenience.CompletionStatusUnprocessed)
	s.NoError(err)
	s.Nil(input2)

	input2, err = s.inputRepository.FindByStatus(ctx, convenience.CompletionStatusAccepted)
	s.NoError(err)
	s.Equal(2222, input2.Index)
}

func (s *InputRepositorySuite) TestFindByIndexGt() {
	defer s.teardown()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.Address{},
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    common.Address{},
		})
		s.NoError(err)
		s.Equal(i, input.Index)
	}
	filters := []*convenience.ConvenienceFilter{}
	value := "1"
	field := INDEX_FIELD
	filters = append(filters, &convenience.ConvenienceFilter{
		Field: &field,
		Gt:    &value,
	})
	resp, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(3, int(resp.Total))
}

func (s *InputRepositorySuite) TestFindByIndexLt() {
	defer s.teardown()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.Address{},
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    common.Address{},
		})
		s.NoError(err)
		s.Equal(i, input.Index)
	}
	filters := []*convenience.ConvenienceFilter{}
	value := "3"
	field := INDEX_FIELD
	filters = append(filters, &convenience.ConvenienceFilter{
		Field: &field,
		Lt:    &value,
	})
	resp, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(3, int(resp.Total))
}

func (s *InputRepositorySuite) TestFindByMsgSender() {
	defer s.teardown()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.HexToAddress(fmt.Sprintf("000000000000000000000000000000000000000%d", i)),
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    common.Address{},
		})
		s.NoError(err)
		s.Equal(i, input.Index)
	}
	filters := []*convenience.ConvenienceFilter{}
	value := "0x0000000000000000000000000000000000000002"
	field := "MsgSender"
	filters = append(filters, &convenience.ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	})
	resp, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(1, int(resp.Total))
	s.Equal(common.HexToAddress(value), resp.Rows[0].MsgSender)
}

func (s *InputRepositorySuite) TestColumnDappAddressExists() {
	query := `PRAGMA table_info(convenience_inputs);`

	rows, err := s.inputRepository.Db.Queryx(query)
	s.NoError(err)

	defer rows.Close()

	var columnExists bool
	for rows.Next() {
		var cid int
		var name, fieldType string
		var notNull, pk int
		var dfltValue interface{}

		err = rows.Scan(&cid, &name, &fieldType, &notNull, &dfltValue, &pk)
		s.NoError(err)

		if name == "app_contract" {
			columnExists = true
			break
		}
	}

	s.True(columnExists, "Column 'app_contract' does not exist in the table 'convenience_inputs'")

}

func (s *InputRepositorySuite) TestCreateInputAndCheckAppContract() {
	defer s.teardown()
	ctx := context.Background()
	_, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})

	s.NoError(err)

	input2, err := s.inputRepository.FindByIndex(ctx, 2222)
	s.NoError(err)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", input2.AppContract.Hex())
}

func (s *InputRepositorySuite) TestFindInputByAppContractAndIndex() {
	defer s.teardown()
	ctx := context.Background()
	_, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})
	s.NoError(err)
	_, err = s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          3333,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0xFF22"),
		BlockNumber:    2,
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"),
	})
	s.NoError(err)

	input, err := s.inputRepository.FindInputByAppContractAndIndex(ctx, 3333, common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"))
	s.NoError(err)
	slog.Debug("INPUT: ", "input", input)

	s.Equal(common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"), input.AppContract)
	s.Equal(3333, input.Index)
	s.Equal(uint64(2), input.BlockNumber)
}

func (s *InputRepositorySuite) teardown() {
	defer os.RemoveAll(s.tempDir)
}
