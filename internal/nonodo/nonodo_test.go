// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package nonodo

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gligneul/nonodo/internal/foundry"
	"github.com/gligneul/nonodo/internal/inspect"
	"github.com/gligneul/nonodo/internal/readerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const testTimeout = 5 * time.Second

type NonodoSuite struct {
	suite.Suite
	ctx           context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	graphqlClient graphql.Client
	inspectClient *inspect.ClientWithResponses
}

//
// Test Cases
//

func (s *NonodoSuite) TestItProcessesAdvanceInputs() {
	opts := NewNonodoOpts()
	opts.BuiltInEcho = true
	s.SetupTest(opts)

	s.T().Log("sending advance inputs")
	const n = 3
	payloads := make([][]byte, n)
	for i := 0; i < n; i++ {
		payloads[i] = s.makePayload()
		err := foundry.AddInput(s.ctx, payloads[i])
		s.Require().Nil(err)
	}

	s.T().Log("waiting until last input is ready")
	err := s.waitForAdvanceInput(n - 1)
	s.Require().Nil(err)

	s.T().Log("verifying node state")
	response, err := readerclient.State(s.ctx, s.graphqlClient)
	s.Require().Nil(err)
	for i := 0; i < n; i++ {
		input := response.Inputs.Edges[i].Node
		s.Equal(i, input.Index)
		s.Equal(payloads[i], s.decodeHex(input.Payload))
		s.Equal(payloads[i], s.decodeHex(input.Payload))
		s.Equal(foundry.SenderAddress[:], s.decodeHex(input.MsgSender))
		voucher := input.Vouchers.Edges[0].Node
		s.Equal(payloads[i], s.decodeHex(voucher.Payload))
		s.Equal(foundry.SenderAddress[:], s.decodeHex(voucher.Destination))
		s.Equal(payloads[i], s.decodeHex(input.Notices.Edges[0].Node.Payload))
		s.Equal(payloads[i], s.decodeHex(input.Reports.Edges[0].Node.Payload))
	}
}

func (s *NonodoSuite) TestItProcessesInspectInputs() {
	opts := NewNonodoOpts()
	opts.BuiltInEcho = true
	s.SetupTest(opts)

	s.T().Log("sending inspect inputs")
	const n = 3
	for i := 0; i < n; i++ {
		payload := s.makePayload()
		response, err := s.sendInspect(payload)
		s.Nil(err)
		s.Equal(http.StatusOK, response.StatusCode())
		s.Equal("0x", response.JSON200.ExceptionPayload)
		s.Equal(0, response.JSON200.ProcessedInputCount)
		s.Len(response.JSON200.Reports, 1)
		s.Equal(payload, s.decodeHex(response.JSON200.Reports[0].Payload))
		s.Equal(inspect.Accepted, response.JSON200.Status)
	}
}

func (s *NonodoSuite) TestItWorksWithExternalApplication() {
	opts := NewNonodoOpts()
	opts.ApplicationArgs = []string{
		"go",
		"run",
		"github.com/gligneul/nonodo/internal/echoapp/echoapp",
		"--endpoint",
		fmt.Sprintf("http://%v:%v/rollup", opts.HttpAddress, opts.HttpPort),
	}
	s.SetupTest(opts)

	s.T().Log("sending inspect to external application")
	payload := s.makePayload()

	response, err := s.sendInspect(payload)
	s.Require().Nil(err)
	s.Require().Equal(http.StatusOK, response.StatusCode())
	s.Require().Equal(payload, s.decodeHex(response.JSON200.Reports[0].Payload))
}

func TestItFailsToStartWhenThereIsApplicationConflict(t *testing.T) {
	// This test doesn't use the suite because the worker fails imediatelly.
	opts := NewNonodoOpts()
	opts.BuiltInEcho = true
	opts.ApplicationArgs = []string{"test"}
	_, err := NewNonodoWorker(opts)
	assert.ErrorIs(t, ApplicationConflictErr, err)
}

//
// Setup and tear down
//

// Setup the nonodo suite.
// This method requires the nonodo options, so each test must call it explicitly.
func (s *NonodoSuite) SetupTest(opts NonodoOpts) {
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	var workerCtx context.Context
	workerCtx, s.workerCancel = context.WithCancel(s.ctx)

	w, err := NewNonodoWorker(opts)
	s.Nil(err)

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

	graphqlEndpoint := fmt.Sprintf("http://%v:%v/graphql", opts.HttpAddress, opts.HttpPort)
	s.graphqlClient = graphql.NewClient(graphqlEndpoint, nil)

	inspectEndpoint := fmt.Sprintf("http://%v:%v/inspect", opts.HttpAddress, opts.HttpPort)
	s.inspectClient, err = inspect.NewClientWithResponses(inspectEndpoint)
	s.Nil(err)
}

func (s *NonodoSuite) TearDownTest() {
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Nil(err)
	}
	s.timeoutCancel()
}

//
// Helper functions
//

// Wait for the given input to be ready.
func (s *NonodoSuite) waitForAdvanceInput(inputIndex int) error {
	const pollRetries = 100
	const pollInterval = 10 * time.Millisecond
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
func (s *NonodoSuite) makePayload() []byte {
	payload := make([]byte, 32)
	_, err := rand.Read(payload)
	s.Require().Nil(err)
	return payload
}

// Decode the hex string into bytes.
func (s *NonodoSuite) decodeHex(value string) []byte {
	bytes, err := hexutil.Decode(value)
	s.Require().Nil(err)
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
	suite.Run(t, &NonodoSuite{})
}
