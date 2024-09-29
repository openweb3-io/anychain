package types

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	wallet_common "github.com/openweb3-io/anychain/pkg/ethereum/common"
)

var (
	// ErrInvalidSendTxArgs is returned when the structure of SendTxArgs is ambigious.
	ErrInvalidSendTxArgs = errors.New("transaction arguments are invalid")
	// ErrUnexpectedArgs is returned when args are of unexpected length.
	ErrUnexpectedArgs = errors.New("unexpected args")
	//ErrInvalidTxSender is returned when selected account is different than From field.
	ErrInvalidTxSender = errors.New("transaction can only be send by its creator")
	//ErrAccountDoesntExist is sent when provided sub-account is not stored in database.
	ErrAccountDoesntExist = errors.New("account doesn't exist")

	// ErrInvalidSignatureSize is returned if a signature is not 65 bytes to avoid panic from go-ethereum
	ErrInvalidSignatureSize = errors.New("signature size must be 65")
)

type ErrBadNonce struct {
	Nonce         uint64
	ExpectedNonce uint64
}

func (e *ErrBadNonce) Error() string {
	return fmt.Sprintf("bad nonce. expected %d, got %d", e.ExpectedNonce, e.Nonce)
}

type SendTxArgs struct {
	From                 common.Address  `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Input                []byte          `json:"input"`
	Data                 []byte          `json:"data"`
	// additional data
	MultiTransactionID wallet_common.MultiTransactionIDType
	Symbol             string
}

// IsDynamicFeeTx checks whether dynamic fee parameters are set for the tx
func (args SendTxArgs) IsDynamicFeeTx() bool {
	return args.MaxFeePerGas != nil && args.MaxPriorityFeePerGas != nil
}

// GetInput returns either Input or Data field's value dependent on what is filled.
func (args SendTxArgs) GetInput() []byte {
	if len(args.Input) > 0 {
		return args.Input
	}

	return args.Data
}

func (args SendTxArgs) Valid() bool {
	if len(args.Input) == 0 || len(args.Data) == 0 {
		return true
	}

	return bytes.Equal(args.Input, args.Data)
}

func (args SendTxArgs) ToTransactOpts(signerFn bind.SignerFn) *bind.TransactOpts {
	var gasFeeCap *big.Int
	if args.MaxFeePerGas != nil {
		gasFeeCap = (*big.Int)(args.MaxFeePerGas)
	}

	var gasTipCap *big.Int
	if args.MaxPriorityFeePerGas != nil {
		gasTipCap = (*big.Int)(args.MaxPriorityFeePerGas)
	}

	var nonce *big.Int
	if args.Nonce != nil {
		nonce = new(big.Int).SetUint64((uint64)(*args.Nonce))
	}

	var gasPrice *big.Int
	if args.GasPrice != nil {
		gasPrice = (*big.Int)(args.GasPrice)
	}

	var gasLimit uint64
	if args.Gas != nil {
		gasLimit = uint64(*args.Gas)
	}

	return &bind.TransactOpts{
		From:      common.Address(args.From),
		Signer:    signerFn,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     nonce,
	}
}
