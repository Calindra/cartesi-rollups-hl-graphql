// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package nonodo

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/commons"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/devnet"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/inspect"
	"github.com/calindra/cartesi-rollups-hl-graphql/internal/readerclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/suite"
)

const testTimeout = 5 * time.Second

type NonodoSuite struct {
	suite.Suite
	ctx           context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	rpcUrl        string
	graphqlClient graphql.Client
	inspectClient *inspect.ClientWithResponses
	nonce         int
}

//
// Test Cases
//

func (s *NonodoSuite) TestItProcessesAdvanceInputs() {
	// defer s.teardown()
	opts := NewNonodoOpts()
	// we are using file cuz there is problem with memory
	// no such table: reports
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	opts.EnableEcho = true
	opts.SqliteFile = path.Join(tempDir, sqliteFileName)
	s.setupTest(opts)
	defer os.RemoveAll(tempDir)
	s.T().Log("sending advance inputs")
	const n = 3
	var payloads [n][32]byte
	for i := 0; i < n; i++ {
		payloads[i] = s.makePayload()
		err := devnet.AddInput(s.ctx, s.rpcUrl, payloads[i][:])
		s.NoError(err)
	}

	s.T().Log("waiting until last input is ready")
	err = s.waitForAdvanceInput(n - 1)
	s.NoError(err)

	s.T().Log("verifying node state")
	response, err := readerclient.State(s.ctx, s.graphqlClient)
	s.NoError(err)
	for i := 0; i < n; i++ {
		input := response.Inputs.Edges[i].Node
		s.Equal(i, input.Index)
		s.Equal(payloads[i][:], s.decodeHex(input.Payload))
		s.Equal(devnet.SenderAddress, input.MsgSender)
		voucher := input.Vouchers.Edges[0].Node
		s.Equal(payloads[i][:], s.decodeHex(voucher.Payload))
		s.Equal(devnet.SenderAddress, voucher.Destination)
		s.Equal(common.Bytes2Hex(payloads[i][:])+"ff",
			input.Notices.Edges[0].Node.Payload[2:])
		s.Equal(payloads[i][:], s.decodeHex(input.Reports.Edges[0].Node.Payload))
	}
}

func (s *NonodoSuite) TestGraphQLVoucher() {
	opts := NewNonodoOpts()
	// we are using file cuz there is problem with memory
	// no such table: reports
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	opts.EnableEcho = true
	opts.SqliteFile = path.Join(tempDir, sqliteFileName)
	s.setupTest(opts)
	defer os.RemoveAll(tempDir)
	s.T().Log("sending advance inputs")
	const n = 3
	var payloads [n][32]byte
	for i := 0; i < n; i++ {
		payloads[i] = s.makePayload()
		err := devnet.AddInput(s.ctx, s.rpcUrl, payloads[i][:])
		s.NoError(err)
	}

	s.T().Log("waiting until last input is ready")
	err = s.waitForAdvanceInput(n - 1)
	s.NoError(err)

	s.T().Log("verifying voucher")
	voucher, err := readerclient.GetVoucher(s.ctx, s.graphqlClient, 0, 1)
	s.NoError(err)
	s.Equal(payloads[1][:], s.decodeHex(voucher.Voucher.Payload))
}

func (s *NonodoSuite) TestGraphQLNotice() {
	opts := NewNonodoOpts()
	// we are using file cuz there is problem with memory
	// no such table: reports
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	opts.EnableEcho = true
	opts.SqliteFile = path.Join(tempDir, sqliteFileName)
	s.setupTest(opts)
	defer os.RemoveAll(tempDir)
	s.T().Log("sending advance inputs")
	const n = 3
	var payloads [n][32]byte
	for i := 0; i < n; i++ {
		payloads[i] = s.makePayload()
		err := devnet.AddInput(s.ctx, s.rpcUrl, payloads[i][:])
		s.NoError(err)
	}

	s.T().Log("waiting until last input is ready")
	err = s.waitForAdvanceInput(n - 1)
	s.NoError(err)

	s.T().Log("verifying voucher")
	notice, err := readerclient.GetNotice(s.ctx, s.graphqlClient, 0, 1)
	s.NoError(err)
	s.Equal(common.Bytes2Hex(payloads[1][:])+"ff",
		notice.Notice.Payload[2:])
}

func (s *NonodoSuite) TestItProcessesInspectInputs() {
	// defer s.teardown()
	opts := NewNonodoOpts()
	opts.EnableEcho = true
	s.setupTest(opts)

	s.T().Log("sending inspect inputs")
	const n = 3
	for i := 0; i < n; i++ {
		payload := s.makePayload()
		response, err := s.sendInspect(payload[:])
		s.NoError(err)
		s.Equal(http.StatusOK, response.StatusCode())
		s.Equal("0x", response.JSON200.ExceptionPayload)
		s.Equal(0, response.JSON200.ProcessedInputCount)
		s.Len(response.JSON200.Reports, 1)
		s.Equal(payload[:], s.decodeHex(response.JSON200.Reports[0].Payload))
		s.Equal(inspect.Accepted, response.JSON200.Status)
	}
}

//
// Setup and tear down
//

// Setup the nonodo suite.
// This method requires the nonodo options, so each test must call it explicitly.
func (s *NonodoSuite) setupTest(opts NonodoOpts) {
	s.nonce += 1
	opts.AnvilPort += s.nonce
	opts.HttpPort += s.nonce + 100
	s.T().Log("ports", "http", opts.HttpPort, "anvil", opts.AnvilPort)
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	var workerCtx context.Context
	workerCtx, s.workerCancel = context.WithCancel(s.ctx)

	w := NewSupervisor(opts)

	ready := make(chan struct{})
	go func() {
		s.workerResult <- w.Start(workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("nonodo ready")
	}

	s.rpcUrl = fmt.Sprintf("http://127.0.0.1:%v", opts.AnvilPort)

	graphqlEndpoint := fmt.Sprintf("http://%s:%v/graphql", opts.HttpAddress, opts.HttpPort)
	s.graphqlClient = graphql.NewClient(graphqlEndpoint, nil)

	inspectEndpoint := fmt.Sprintf("http://%s:%v/", opts.HttpAddress, opts.HttpPort)
	var err error
	s.inspectClient, err = inspect.NewClientWithResponses(inspectEndpoint)
	s.NoError(err)
}

func (s *NonodoSuite) TearDownTest() {
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
	s.T().Log("teardown ok.")
}

//
// Helper functions
//

// Wait for the given input to be ready.
func (s *NonodoSuite) waitForAdvanceInput(inputIndex int) error {
	const pollRetries = 100
	const pollInterval = 15 * time.Millisecond
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < pollRetries; i++ {
		result, err := readerclient.InputStatus(s.ctx, s.graphqlClient, inputIndex)
		if err != nil && !strings.Contains(err.Error(), "input not found") {
			return fmt.Errorf("failed to get input status: %w", err)
		}
		if result.Input.Status == readerclient.CompletionStatusAccepted {
			return nil
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("input never got ready")
}

// Create a random payload to use in the tests
func (s *NonodoSuite) makePayload() [32]byte {
	var payload [32]byte
	_, err := rand.Read(payload[:])
	s.NoError(err)
	fmt.Println(payload)
	return payload
}

// Decode the hex string into bytes.
func (s *NonodoSuite) decodeHex(value string) []byte {
	bytes, err := hexutil.Decode(value)
	s.NoError(err)
	return bytes
}

// Send an inspect request with the given payload.
func (s *NonodoSuite) sendInspect(payload []byte) (*inspect.InspectPostResponse, error) {
	return s.inspectClient.InspectPostWithBodyWithResponse(
		s.ctx,
		"application/octet-stream",
		bytes.NewReader(payload),
	)
}

//
// Suite entry point
//

func TestNonodoSuite(t *testing.T) {
	suite.Run(t, &NonodoSuite{nonce: 0})
}
