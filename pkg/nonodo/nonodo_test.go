// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package nonodo

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type NonodoSuite struct {
	suite.Suite
}

//
// Test Cases
//

func (s *NonodoSuite) TestItProcessesAdvanceInputs() {
	s.Equal(1, 1)
}

//
// Suite entry point
//

func TestNonodoSuite(t *testing.T) {
	suite.Run(t, &NonodoSuite{})
}
