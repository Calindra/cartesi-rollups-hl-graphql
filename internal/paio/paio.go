package paio

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/labstack/echo/v4"
)

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ./oapi-paio.yaml

//go:embed paio.json
var DEFINITION string

type PaioTypedata struct {
	apitypes.TypedData
	Account common.Address `json:"account"`
}

type PaioAPI struct {
	availClient     *avail.AvailClient
	inputRepository *repository.InputRepository
}

// SendTransaction implements ServerInterface.
func (p *PaioAPI) SendTransaction(ctx echo.Context) error {
	var request SendTransactionJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return err
	}
	slog.Debug("Sending Avail transaction", "request", request)
	sigAndData := commons.SigAndData{
		Signature: request.Signature,
		TypedData: request.TypedData,
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		slog.Error("Error json.Marshal message:", "err", err)
		return err
	}
	hash, err := p.availClient.DefaultSubmit(stdCtx, string(jsonPayload))
	if err != nil {
		slog.Error("Error DefaultSubmit message:", "err", err)
		return err
	}
	_ = ctx.String(http.StatusOK, hash.Hex())
	return nil
}

func (p *PaioAPI) GetNonce(ctx echo.Context) error {
	var request GetNonceJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if request.MsgSender == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "msg_sender is required"})
	}

	filters := []*model.ConvenienceFilter{}
	msgSenderField := "MsgSender"
	filters = append(filters, &model.ConvenienceFilter{
		Field: &msgSenderField,
		Eq:    &request.MsgSender,
	})

	typeField := "Type"
	inputBoxType := "inputbox"
	filters = append(filters, &model.ConvenienceFilter{
		Field: &typeField,
		Ne:    &inputBoxType,
	})
	inputs, err := p.inputRepository.FindAll(stdCtx, nil, nil, nil, nil, filters)

	if err != nil {
		slog.Error("Error querying for inputs:", "err", err)
		return err
	}

	nonce := int(inputs.Total + 1)
	response := NonceResponse{
		Nonce: &nonce,
	}

	return ctx.JSON(http.StatusOK, response)
}

func (p *PaioAPI) SaveTransaction(ctx echo.Context) error {
	var request SaveTransactionJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	if request.Signature == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "signature is required"})
	}

	if request.Message == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "message is required"})
	}

	// decode the ABI from message
	// https://github.com/fabiooshiro/frontend-web-cartesi/blob/16913e945ef687bd07b6c3900d63cb23d69390b1/src/Input.tsx#L195C13-L212C15
	decoder, err := abi.JSON(strings.NewReader(DEFINITION))
	if err != nil {
		slog.Error("error decoding ABI:", "err", err)
		return ctx.JSON(http.StatusInternalServerError, echo.Map{"error": "avail: error decoding ABI"})
	}
	method, ok := decoder.Methods["signingMessage"]
	if !ok {
		slog.Error("error getting method signingMessage", "err", err)
		return ctx.JSON(http.StatusInternalServerError, echo.Map{"error": "avail: error getting method signingMessage"})
	}

	// decode the message, message dont have 4 bytes of method id
	message := common.Hex2Bytes(strings.TrimPrefix(request.Message, "0x"))
	data := make(map[string]any)
	err = method.Inputs.UnpackIntoMap(data, message)
	if err != nil {
		slog.Error("error unpacking message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error unpacking message"})
	}

	// Validate the data from the message
	app, ok := data["app"].(common.Address)
	if !ok {
		slog.Error("error extracting app from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting app from message"})
	}
	nonce, ok := data["nonce"].(uint64)
	if !ok {
		slog.Error("error extracting nonce from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting nonce from message"})
	}
	maxGasPrice, ok := data["max_gas_price"].(*big.Int)
	if !ok {
		slog.Error("error extracting max_gas_price from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting max_gas_price from message"})
	}
	dataBytes, ok := data["data"].([]byte)
	if !ok {
		slog.Error("error extracting data from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting data from message"})
	}

	// fill the typedData
	// https://github.com/fabiooshiro/frontend-web-cartesi/blob/16913e945ef687bd07b6c3900d63cb23d69390b1/src/Input.tsx#L65
	chainId := 11155111 // Paio's fixed value for Anvil and Hardhat
	verifyingContract := common.HexToAddress("0x0")

	var typedata PaioTypedata
	typedata.Account = common.Address{}
	typedata.Domain = apitypes.TypedDataDomain{
		Name:              "CartesiPaio",
		Version:           "0.0.1",
		ChainId:           math.NewHexOrDecimal256(int64(chainId)),
		VerifyingContract: verifyingContract.String(),
	}
	typedata.Types = apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"CartesiMessage": {
			{Name: "app", Type: "address"},
			{Name: "nonce", Type: "uint64"},
			{Name: "max_gas_price", Type: "uint128"},
			{Name: "data", Type: "bytes"},
		}}
	typedata.PrimaryType = "CartesiMessage"
	typedata.Message = apitypes.TypedDataMessage{
		"app":           app.String(),
		"nonce":         nonce,
		"max_gas_price": maxGasPrice.String(),
		"data":          fmt.Sprintf("0x%s", common.Bytes2Hex(dataBytes)),
	}

	typeJSON, err := json.Marshal(typedata)
	if err != nil {
		return fmt.Errorf("error marshalling typedata: %w", err)
	}

	// set the typedData as string json below
	sigAndData := commons.SigAndData{
		Signature: request.Signature,
		TypedData: base64.StdEncoding.EncodeToString(typeJSON),
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		slog.Error("Error json.Marshal message:", "err", err)
		return err
	}
	slog.Debug("SaveTransaction", "jsonPayload", string(jsonPayload))
	msgSender, _, signature, err := commons.ExtractSigAndData(string(jsonPayload))

	if err != nil {
		slog.Error("Error:", "err", err)
		return err
	}

	dappAddress := app.String()
	payload := string(dataBytes)

	slog.Debug("Save input",
		"dappAddress", dappAddress,
		"msgSender", msgSender,
		"nonce", nonce,
		"maxGasPrice", maxGasPrice,
		"payload", payload,
	)

	payloadBytes := []byte(payload)
	if strings.HasPrefix(payload, "0x") {
		payload = payload[2:] // remove 0x
		payloadBytes, err = hex.DecodeString(payload)
		if err != nil {
			return err
		}
	}

	inputCount, err := p.inputRepository.Count(stdCtx, nil)

	if err != nil {
		slog.Error("Error counting inputs:", "err", err)
		return err
	}

	createdInput, err := p.inputRepository.Create(stdCtx, model.AdvanceInput{
		Index:                int(inputCount + 1),
		CartesiTransactionId: common.Bytes2Hex(crypto.Keccak256(signature)),
		MsgSender:            msgSender,
		Payload:              payloadBytes,
		AppContract:          common.HexToAddress(dappAddress),
		InputBoxIndex:        -2,
		Type:                 "Avail",
	})

	if err != nil {
		slog.Error("Error creating inputs:", "err", err)
		return err
	}

	transactionId := fmt.Sprintf("%d", createdInput.Index)

	response := TransactionResponse{
		Id: &transactionId,
	}

	return ctx.JSON(http.StatusOK, response)
}

// Register the Paio API to echo
func Register(e *echo.Echo, availClient *avail.AvailClient, inputRepository *repository.InputRepository) {
	var paioAPI ServerInterface = &PaioAPI{availClient, inputRepository}
	RegisterHandlers(e, paioAPI)
}
