package erc20

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	USDT_CONTRACT_ADDRESS = "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	USDT_CONTRACT_NAME    = "ERC20"
	USDT_TOKEN_NAME       = "USDT"
)

var (
	UsdtContract *ERC20Contract
)

func init() {
	UsdtContract = &ERC20Contract{
		Name:      USDT_CONTRACT_NAME,
		TokenName: USDT_TOKEN_NAME,
		Address:   USDT_CONTRACT_ADDRESS,
		Abi:       IERC20MetaData.ABI,
	}
}

type ERC20Contract struct {
	Address   string
	Name      string
	TokenName string
	Abi       string
}

func (e *ERC20Contract) GetContractAddress() string {
	return e.Address
}

func (e *ERC20Contract) GetContractName() string {
	return e.Name
}

func (e *ERC20Contract) GetContractAbi() string {
	return e.Abi
}

func (e *ERC20Contract) GetTokenName() string {
	return e.TokenName
}

func (e *ERC20Contract) ParseTransfer(log *types.Log) (*IERC20Transfer, error) {
	filter, err := NewIERC20Filterer(common.HexToAddress(e.GetContractAddress()), nil)
	if err != nil {
		return nil, err
	}

	return filter.ParseTransfer(*log)
}
