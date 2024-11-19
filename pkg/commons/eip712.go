package commons

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type SigAndData struct {
	Signature string `json:"signature"`
	TypedData string `json:"typedData"`
}

const (
	HARDHAT         = 31337
	PURPOSE_INDEX   = 44
	COIN_TYPE_INDEX = 60
)

func NewCartesiDomain(chainId *math.HexOrDecimal256) apitypes.TypedDataDomain {
	verifyingContract := common.HexToAddress("0x0")
	return apitypes.TypedDataDomain{
		Name:              "Cartesi",
		Version:           "0.1.0",
		ChainId:           chainId,
		VerifyingContract: verifyingContract.String(),
	}
}

// Implement the hashing function based on EIP-712 requirements
func HashEIP712Message(data apitypes.TypedData) ([]byte, error) {
	hash, _, err := apitypes.TypedDataAndHash(data)
	if err != nil {
		return []byte(""), err
	}
	return hash, nil
}

// Sign the hash with the private key
func SignMessage(hash []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func trimNonPrintablePrefix(s string) string {
	for i, r := range s {
		if unicode.IsPrint(r) && r == '{' {
			return s[i:] // Return the substring starting from the first printable character
		}
	}
	return "" // Return empty string if no printable character is found
}

func ExtractSigAndData(raw string) (common.Address, apitypes.TypedData, []byte, error) {
	var sigAndData SigAndData
	if err := json.Unmarshal([]byte(trimNonPrintablePrefix(raw)), &sigAndData); err != nil {
		slog.Error("unmarshal error", "error", err)
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("unmarshal sigAndData: %w", err)
	}

	signature, err := hexutil.Decode(sigAndData.Signature)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("decode signature: %w", err)
	}

	typedDataBytes, err := base64.StdEncoding.DecodeString(sigAndData.TypedData)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("decode typed data: %w", err)
	}
	slog.Debug("ExtractSigAndData", "typedDataBytes", string(typedDataBytes))
	typedData := apitypes.TypedData{}
	if err := json.Unmarshal(typedDataBytes, &typedData); err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("unmarshal typed data: %w", err)
	}

	slog.Debug("ExtractSigAndData", "typedData", typedData.Message["app"])
	dataHash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("typed data hash: %w", err)
	}

	// update the recovery id
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	signature[64] -= 27

	// get the pubkey used to sign this signature
	sigPubkey, err := crypto.Ecrecover(dataHash, signature)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("ecrecover: %w", err)
	}
	// fmt.Printf("SigPubkey: %s\n", common.Bytes2Hex(sigPubkey))
	pubkey, err := crypto.UnmarshalPubkey(sigPubkey)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, []byte{}, fmt.Errorf("unmarshal: %w", err)
	}
	address := crypto.PubkeyToAddress(*pubkey)
	slog.Debug("ExtractSigAndData", "publicKeyAddress", address)
	return address, typedData, signature, nil
}
