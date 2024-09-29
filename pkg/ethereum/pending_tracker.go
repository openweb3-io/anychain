package ethereum

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	wallet_common "github.com/openweb3-io/anychain/pkg/ethereum/common"
)

type TxStatus = string

// Values for status column in pending_transactions
const (
	Pending TxStatus = "Pending"
	Success TxStatus = "Success"
	Failed  TxStatus = "Failed"
)

type AutoDeleteType = bool

const (
	AutoDelete AutoDeleteType = true
	Keep       AutoDeleteType = false
)

type PendingTrxType string

const (
	RegisterENS               PendingTrxType = "RegisterENS"
	ReleaseENS                PendingTrxType = "ReleaseENS"
	SetPubKey                 PendingTrxType = "SetPubKey"
	BuyStickerPack            PendingTrxType = "BuyStickerPack"
	WalletTransfer            PendingTrxType = "WalletTransfer"
	DeployCommunityToken      PendingTrxType = "DeployCommunityToken"
	AirdropCommunityToken     PendingTrxType = "AirdropCommunityToken"
	RemoteDestructCollectible PendingTrxType = "RemoteDestructCollectible"
	BurnCommunityToken        PendingTrxType = "BurnCommunityToken"
	DeployOwnerToken          PendingTrxType = "DeployOwnerToken"
	SetSignerPublicKey        PendingTrxType = "SetSignerPublicKey"
	WalletConnectTransfer     PendingTrxType = "WalletConnectTransfer"
)

type PendingTransaction struct {
	Hash               common.Hash                          `json:"hash"`
	Timestamp          uint64                               `json:"timestamp"`
	Value              *big.Int                             `json:"value"`
	From               common.Address                       `json:"from"`
	To                 common.Address                       `json:"to"`
	Data               string                               `json:"data"`
	Symbol             string                               `json:"symbol"`
	GasPrice           *big.Int                             `json:"gasPrice"`
	GasLimit           *big.Int                             `json:"gasLimit"`
	Type               PendingTrxType                       `json:"type"`
	AdditionalData     string                               `json:"additionalData"`
	ChainID            wallet_common.ChainID                `json:"network_id"`
	MultiTransactionID wallet_common.MultiTransactionIDType `json:"multi_transaction_id"`
	Nonce              uint64                               `json:"nonce"`

	// nil will insert the default value (Pending) in DB
	Status *TxStatus `json:"status,omitempty"`
	// nil will insert the default value (true) in DB
	AutoDelete *bool `json:"autoDelete,omitempty"`
}

type IPendingTxTracker interface {
	CountPendingTxsFromNonce(chainID wallet_common.ChainID, address common.Address, nonce uint64) (pendingTx uint64, err error)
	StoreAndTrackPendingTx(pendingTx *PendingTransaction) error
}

type NoopPendingTxTracker struct {
}

func (tm *NoopPendingTxTracker) CountPendingTxsFromNonce(chainID wallet_common.ChainID, address common.Address, nonce uint64) (pendingTx uint64, err error) {
	return 1, nil
}

func (tm *NoopPendingTxTracker) StoreAndTrackPendingTx(pendingTx *PendingTransaction) error {
	// TODO
	return nil
}
