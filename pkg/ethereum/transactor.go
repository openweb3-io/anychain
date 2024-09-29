package ethereum

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	wallet_common "github.com/openweb3-io/anychain/pkg/ethereum/common"
	_types "github.com/openweb3-io/anychain/pkg/ethereum/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	// sendTxTimeout defines how many seconds to wait before returning result in sentTransaction().
	sendTxTimeout = 300 * time.Second

	defaultGas = 90000

	ValidSignatureSize = 65
)

type ITransactor interface {
	NextNonce(
		ctx context.Context,
		chainID *big.Int,
		from common.Address,
	) (uint64, error)
	EstimateGas(
		ctx context.Context,
		from common.Address,
		to common.Address,
		value *big.Int,
		input []byte,
	) (uint64, error)
	SendTransaction(
		ctx context.Context,
		sendArgs _types.SendTxArgs,
		signer Signer,
		lastUsedNonce int64,
	) (hash _types.Hash, nonce uint64, err error)
	SendTransactionWithChainID(
		ctx context.Context,
		chainID uint64,
		sendArgs _types.SendTxArgs,
		signer Signer,
		lastUsedNonce int64,
	) (hash _types.Hash, nonce uint64, err error)
	ValidateAndBuildTransaction(
		ctx context.Context,
		chainID uint64,
		sendArgs _types.SendTxArgs,
		lastUsedNonce int64,
	) (tx *types.Transaction, nonce uint64, err error)

	AddSignatureToTransaction(
		chainID uint64,
		tx *types.Transaction,
		sig []byte,
	) (*types.Transaction, error)
	SendRawTransaction(
		ctx context.Context,
		chainID uint64,
		rawTx string,
	) error
	BuildTransactionWithSignature(
		ctx context.Context,
		chainID uint64,
		args _types.SendTxArgs,
		sig []byte,
	) (*types.Transaction, error)
	SendTransactionWithSignature(
		ctx context.Context,
		from common.Address,
		symbol string,
		multiTransactionID wallet_common.MultiTransactionIDType,
		tx *types.Transaction,
	) (hash _types.Hash, err error)
}

type Transactor struct {
	chainId        *big.Int
	client         *ethclient.Client
	pendingTracker IPendingTxTracker
}

func NewTransactor(
	client *ethclient.Client,
	chainId *big.Int,
	pendingTracker IPendingTxTracker,
) *Transactor {
	if pendingTracker == nil {
		pendingTracker = &NoopPendingTxTracker{}
	}

	return &Transactor{
		chainId:        chainId,
		client:         client,
		pendingTracker: pendingTracker,
	}
}

func (t *Transactor) NextNonce(ctx context.Context, chainID *big.Int, from common.Address) (uint64, error) {
	nonce, err := t.client.PendingNonceAt(ctx, common.Address(from))
	if err != nil {
		return 0, err
	}

	var chID uint64
	if chainID != nil {
		chID = chainID.Uint64()
	}

	// We need to take into consideration all pending transactions in case of Optimism, cause the network returns always
	// the nonce of last executed tx + 1 for the next nonce value.
	if chID == wallet_common.OptimismMainnet ||
		chID == wallet_common.OptimismSepolia ||
		chID == wallet_common.OptimismGoerli {
		if t.pendingTracker != nil {
			countOfPendingTXs, err := t.pendingTracker.CountPendingTxsFromNonce(wallet_common.ChainID(chID), common.Address(from), nonce)
			if err != nil {
				return 0, err
			}
			return nonce + countOfPendingTXs, nil
		}
	}

	return nonce, err
}

func (t *Transactor) EstimateGas(
	ctx context.Context,
	from common.Address,
	to common.Address,
	value *big.Int,
	input []byte,
) (uint64, error) {
	return t.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  input,
	})
}

// SendTransaction is an implementation of eth_sendTransaction. It queues the tx to the sign queue.
func (t *Transactor) SendTransaction(
	ctx context.Context,
	sendArgs _types.SendTxArgs,
	signer Signer,
	lastUsedNonce int64,
) (hash _types.Hash, nonce uint64, err error) {
	hash, nonce, err = t.validateAndPropagate(ctx, big.NewInt(1), signer, sendArgs, lastUsedNonce)
	return
}

func (t *Transactor) SendTransactionWithChainID(
	ctx context.Context,
	chainID *big.Int,
	sendArgs _types.SendTxArgs,
	signer Signer,
	lastUsedNonce int64,
) (hash _types.Hash, nonce uint64, err error) {
	hash, nonce, err = t.validateAndPropagate(ctx, chainID, signer, sendArgs, lastUsedNonce)
	return
}

func (t *Transactor) ValidateAndBuildTransaction(
	ctx context.Context,
	chainID *big.Int,
	sendArgs _types.SendTxArgs,
	lastUsedNonce int64,
) (tx *types.Transaction, nonce uint64, err error) {
	tx, err = t.validateAndBuildTransaction(ctx, chainID, sendArgs, lastUsedNonce)
	if err != nil {
		return nil, 0, err
	}

	return tx, tx.Nonce(), err
}

func (t *Transactor) AddSignatureToTransaction(
	chainID *big.Int,
	tx *types.Transaction,
	sig []byte,
) (*types.Transaction, error) {
	if len(sig) != ValidSignatureSize {
		return nil, _types.ErrInvalidSignatureSize
	}

	signer := types.NewLondonSigner(chainID)
	txWithSignature, err := tx.WithSignature(signer, sig)
	if err != nil {
		return nil, err
	}

	return txWithSignature, nil
}

func (t *Transactor) SendRawTransaction(ctx context.Context, rawTx string) error {
	return t.client.Client().CallContext(ctx, nil, "eth_sendRawTransaction", rawTx)
}

func createPendingTransaction(
	from common.Address,
	symbol string,
	chainID uint64,
	multiTransactionID wallet_common.MultiTransactionIDType,
	tx *types.Transaction) (pTx *PendingTransaction) {
	pTx = &PendingTransaction{
		Hash:               tx.Hash(),
		Timestamp:          uint64(time.Now().Unix()),
		Value:              tx.Value(),
		From:               from,
		To:                 *tx.To(),
		Nonce:              tx.Nonce(),
		Data:               string(tx.Data()),
		Type:               WalletTransfer,
		ChainID:            wallet_common.ChainID(chainID),
		MultiTransactionID: multiTransactionID,
		Symbol:             symbol,
		AutoDelete:         new(bool),
	}
	// Transaction downloader will delete pending transaction as soon as it is confirmed
	*pTx.AutoDelete = false
	return
}

func (t *Transactor) StoreAndTrackPendingTx(from common.Address, symbol string, chainID uint64, multiTransactionID wallet_common.MultiTransactionIDType, tx *types.Transaction) error {
	if t.pendingTracker == nil {
		return nil
	}

	pTx := createPendingTransaction(from, symbol, chainID, multiTransactionID, tx)
	return t.pendingTracker.StoreAndTrackPendingTx(pTx)
}

func (t *Transactor) sendTransaction(
	ctx context.Context,
	from common.Address,
	symbol string,
	multiTransactionID wallet_common.MultiTransactionIDType,
	tx *types.Transaction,
) (hash _types.Hash, err error) {
	if err := t.client.SendTransaction(ctx, tx); err != nil {
		return hash, err
	}

	err = t.StoreAndTrackPendingTx(from, symbol, tx.ChainId().Uint64(), multiTransactionID, tx)
	if err != nil {
		return hash, err
	}

	return _types.Hash(tx.Hash()), nil
}

func (t *Transactor) SendTransactionWithSignature(
	ctx context.Context,
	from common.Address,
	symbol string,
	multiTransactionID wallet_common.MultiTransactionIDType,
	tx *types.Transaction,
) (hash _types.Hash, err error) {
	return t.sendTransaction(ctx, from, symbol, multiTransactionID, tx)
}

// BuildTransactionAndSendWithSignature receive a transaction and a signature, serialize them together
// It's different from eth_sendRawTransaction because it receives a signature and not a serialized transaction with signature.
// Since the transactions is already signed, we assume it was validated and used the right nonce.
func (t *Transactor) BuildTransactionWithSignature(
	ctx context.Context,
	chainID *big.Int,
	args _types.SendTxArgs,
	sig []byte,
) (*types.Transaction, error) {
	if !args.Valid() {
		return nil, _types.ErrInvalidSendTxArgs
	}

	if len(sig) != ValidSignatureSize {
		return nil, _types.ErrInvalidSignatureSize
	}

	tx := t.buildTransaction(args)
	expectedNonce, err := t.NextNonce(ctx, chainID, args.From)
	if err != nil {
		return nil, err
	}

	if tx.Nonce() != expectedNonce {
		return nil, &_types.ErrBadNonce{Nonce: tx.Nonce(), ExpectedNonce: expectedNonce}
	}

	txWithSignature, err := t.AddSignatureToTransaction(chainID, tx, sig)
	if err != nil {
		return nil, err
	}

	return txWithSignature, nil
}

func (t *Transactor) validateAndBuildTransaction(
	ctx context.Context,
	chainID *big.Int,
	args _types.SendTxArgs,
	lastUsedNonce int64,
) (tx *types.Transaction, err error) {
	if !args.Valid() {
		return nil, _types.ErrInvalidSendTxArgs
	}

	var nonce uint64
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	} else {
		// some chains, like arbitrum doesn't count pending txs in the nonce, so we need to calculate it manually
		if lastUsedNonce < 0 {
			nonce, err = t.client.PendingNonceAt(ctx, args.From)
			if err != nil {
				return nil, err
			}
		} else {
			nonce = uint64(lastUsedNonce) + 1
		}
	}

	gasPrice := (*big.Int)(args.GasPrice)
	// GasPrice should be estimated only for LegacyTx
	if !args.IsDynamicFeeTx() && gasPrice == nil {
		gasPrice, err = t.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to suggest gas price")
		}
	}

	value := (*big.Int)(args.Value)
	var gas uint64
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	} else {
		if args.IsDynamicFeeTx() {
			gasFeeCap := (*big.Int)(args.MaxFeePerGas)
			gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
			gas, err = t.client.EstimateGas(ctx, ethereum.CallMsg{
				From:      args.From,
				To:        args.To,
				GasFeeCap: gasFeeCap,
				GasTipCap: gasTipCap,
				Value:     value,
				Data:      args.GetInput(),
			})
		} else {
			gas, err = t.client.EstimateGas(ctx, ethereum.CallMsg{
				From:     args.From,
				To:       args.To,
				GasPrice: gasPrice,
				Value:    value,
				Data:     args.GetInput(),
			})
		}
		if err != nil {
			return nil, err
		}
	}

	tx = t.buildTransactionWithOverrides(nonce, value, gas, gasPrice, args)
	return tx, nil
}

func (t *Transactor) validateAndPropagate(
	ctx context.Context,
	chainID *big.Int,
	signer Signer,
	args _types.SendTxArgs,
	lastUsedNonce int64,
) (hash _types.Hash, nonce uint64, err error) {
	tx, err := t.validateAndBuildTransaction(ctx, chainID, args, lastUsedNonce)
	if err != nil {
		return hash, nonce, err
	}

	// 计算hash
	eSigner := types.NewLondonSigner(chainID)

	chash := eSigner.Hash(tx)

	// sign
	sig, err := signer.Sign(chash.Bytes())
	if err != nil {
		return hash, nonce, err
	}

	signedTx, err := tx.WithSignature(eSigner, sig)
	if err != nil {
		return hash, nonce, err
	}

	hash, err = t.sendTransaction(ctx, common.Address(args.From), args.Symbol, args.MultiTransactionID, signedTx)
	return hash, tx.Nonce(), err
}

func (t *Transactor) buildTransaction(args _types.SendTxArgs) *types.Transaction {
	var (
		nonce    uint64
		value    *big.Int
		gas      uint64
		gasPrice *big.Int
	)
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}
	if args.Value != nil {
		value = (*big.Int)(args.Value)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if args.GasPrice != nil {
		gasPrice = (*big.Int)(args.GasPrice)
	}

	return t.buildTransactionWithOverrides(nonce, value, gas, gasPrice, args)
}

func (t *Transactor) buildTransactionWithOverrides(
	nonce uint64,
	value *big.Int,
	gas uint64,
	gasPrice *big.Int,
	args _types.SendTxArgs,
) *types.Transaction {
	var tx *types.Transaction

	if args.To != nil {
		var txData types.TxData
		if args.IsDynamicFeeTx() {
			gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
			gasFeeCap := (*big.Int)(args.MaxFeePerGas)

			txData = &types.DynamicFeeTx{
				Nonce:     nonce,
				Gas:       gas,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				To:        args.To,
				Value:     value,
				Data:      args.GetInput(),
			}
		} else {
			txData = &types.LegacyTx{
				Nonce:    nonce,
				GasPrice: gasPrice,
				Gas:      gas,
				To:       args.To,
				Value:    value,
				Data:     args.GetInput(),
			}
		}
		tx = types.NewTx(txData)
		zap.S().Info("New transaction",
			zap.String("From", args.From.String()),
			zap.String("To", args.To.String()),
			zap.Uint64("Gas", gas),
			zap.String("GasPrice", gasPrice.String()),
			zap.String("Value", value.String()),
		)
	} else {
		if args.IsDynamicFeeTx() {
			gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
			gasFeeCap := (*big.Int)(args.MaxFeePerGas)

			txData := &types.DynamicFeeTx{
				Nonce:     nonce,
				Value:     value,
				Gas:       gas,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Data:      args.GetInput(),
			}
			tx = types.NewTx(txData)
		} else {
			tx = types.NewContractCreation(nonce, value, gas, gasPrice, args.GetInput())
		}
		zap.S().Info("New contract",
			zap.String("From", args.From.String()),
			zap.Uint64("Gas", gas),
			zap.String("GasPrice", gasPrice.String()),
			zap.String("Value", value.String()),
			zap.String("Contract address", crypto.CreateAddress(args.From, nonce).String()),
		)
	}

	return tx
}
