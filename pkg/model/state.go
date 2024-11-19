// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"context"
	"fmt"

	cModel "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
)

// Interface that represents the state of the rollup.
type rollupsState interface {

	// Finish the current state, saving the result to the model.
	finish(status cModel.CompletionStatus) error

	// Add voucher to current state.
	addVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error)

	addVoucherWithInput(destination common.Address, payload []byte, inputIndex int) (int, error)

	// Add notice to current state.
	addNotice(payload []byte, appAddress common.Address) (int, error)

	addNoticeWithInput(payload []byte, inputIndex int) (int, error)

	// Add report to current state.
	addReport(appAddress common.Address, payload []byte) error

	// Register exception in current state.
	registerException(payload []byte) error
}

// Convenience OutputDecoder
type Decoder interface {
	HandleOutput(
		ctx context.Context,
		destination common.Address,
		payload string,
		inputIndex uint64,
		outputIndex uint64,
	) error
}

//
// Idle
//

// In the idle state, the model waits for an finish request from the rollups API.
type rollupsStateIdle struct{}

func (s *rollupsStateIdle) finish(status cModel.CompletionStatus) error {
	return nil
}

func (s *rollupsStateIdle) addVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add voucher in idle state")
}

func (s *rollupsStateIdle) addNotice(payload []byte, appAddress common.Address) (int, error) {
	return 0, fmt.Errorf("cannot add notice in current state")
}

func (s *rollupsStateIdle) addReport(appAddress common.Address, payload []byte) error {
	return fmt.Errorf("cannot add report in current state")
}

func (s *rollupsStateIdle) registerException(payload []byte) error {
	return fmt.Errorf("cannot register exception in current state")
}

func (s *rollupsStateIdle) addNoticeWithInput(payload []byte, inputIndex int) (int, error) {
	panic("remove this method please")
}

func (s *rollupsStateIdle) addVoucherWithInput(destination common.Address, payload []byte, inputIndex int) (int, error) {
	panic("remove this method please")
}
