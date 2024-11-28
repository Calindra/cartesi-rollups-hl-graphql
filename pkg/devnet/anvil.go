// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package devnet

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sort"
	"strings"
)

// Default port for the Ethereum node.
const (
	AnvilDefaultAddress = "127.0.0.1"
	AnvilDefaultPort    = 8545
)

// Generate the devnet state and embed it in the Go binary.
//
//go:generate go run ./gen-devnet-state
//go:embed anvil_state.json
var devnetState []byte

//go:embed localhost.json
var localhost []byte

const stateFileName = "anvil_state.json"

const anvilCommand = "anvil"

// Start the anvil process in the host machine.
type AnvilWorker struct {
	Address  string
	Port     int
	Verbose  bool
	AnvilCmd string
}

// Define a struct to represent the structure of your JSON data
type ContractInfo struct {
	Contracts map[string]struct {
		Address string `json:"address"`
	} `json:"contracts"`
}

func GetContractInfo() *ContractInfo {
	var contracts ContractInfo
	if err := json.Unmarshal(localhost, &contracts); err != nil {
		slog.Warn("anvil: failed to unmarshal localhost.json", "error", err)
		return nil
	}
	return &contracts
}

func ShowAddresses() {
	contracts := GetContractInfo()
	var names []string
	for name := range contracts.Contracts {
		names = append(names, name)
	}
	names = append(names, ApplicationContractName)
	contracts.Contracts[ApplicationContractName] = struct {
		Address string "json:\"address\""
	}{
		Address: ApplicationAddress,
	}
	sort.Strings(names)
	space := 28
	addressSpace := 42
	fmt.Printf("%-28s %s\n", "Contract", "Address")
	fmt.Printf("%-28s %s\n", strings.Repeat("─", space), strings.Repeat("─", addressSpace))
	for _, name := range names {
		if contract, ok := contracts.Contracts[name]; ok {
			fmt.Printf("%-28s %s\n", name, contract.Address)
		}
	}
}

// Create a temporary directory with the state file in it.
// The directory should be removed by the callee.
func makeStateTemp() (string, error) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", fmt.Errorf("anvil: failed to create temp dir: %w", err)
	}
	stateFile := path.Join(tempDir, stateFileName)
	const permissions = 0644
	err = os.WriteFile(stateFile, devnetState, permissions)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to write state file: %w", err)
	}
	return tempDir, nil
}