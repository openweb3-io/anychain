package ethereum

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	_types "github.com/openweb3-io/anychain/pkg/ethereum/types"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
)

var (
	Address1 = ""
	Address2 = ""
)

var (
	testGas      = hexutil.Uint64(defaultGas + 1)
	testGasPrice = (*hexutil.Big)(big.NewInt(10))
	testNonce    = hexutil.Uint64(10)
)

func TestTransactorSuite(t *testing.T) {
	suite.Run(t, new(TransactorSuite))
}

type TransactorSuite struct {
	suite.Suite
	transactor *Transactor
}

func (s *TransactorSuite) SetupTest() {
	endpoint := "https://eth-mainnet.public.blastapi.io"

	client, err := ethclient.Dial(endpoint)
	s.Require().NoError(err)

	s.transactor = NewTransactor(
		client,
		nil,
		nil,
	)
}

func (s *TransactorSuite) TearDownTest() {

}

func (s *TransactorSuite) TestGasValues() {
	key, _ := gethcrypto.GenerateKey()
	signer := NewPrivateKeySigner(key)

	testCases := []struct {
		name                 string
		gas                  *hexutil.Uint64
		gasPrice             *hexutil.Big
		maxFeePerGas         *hexutil.Big
		maxPriorityFeePerGas *hexutil.Big
	}{
		{
			"noGasDef",
			nil,
			nil,
			nil,
			nil,
		},
		{
			"gasDefined",
			&testGas,
			nil,
			nil,
			nil,
		},
		{
			"gasPriceDefined",
			nil,
			testGasPrice,
			nil,
			nil,
		},
		{
			"nilSignTransactionSpecificArgs",
			nil,
			nil,
			nil,
			nil,
		},
		{
			"maxFeeAndPriorityset",
			nil,
			nil,
			testGasPrice,
			testGasPrice,
		},
	}

	ctx := context.Background()

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.SetupTest()

			to := common.HexToAddress(Address2)

			args := _types.SendTxArgs{
				From:                 common.HexToAddress(Address1),
				To:                   &to,
				Gas:                  testCase.gas,
				GasPrice:             testCase.gasPrice,
				MaxFeePerGas:         testCase.maxFeePerGas,
				MaxPriorityFeePerGas: testCase.maxPriorityFeePerGas,
			}

			hash, _, err := s.transactor.SendTransaction(ctx, args, signer, -1)
			s.NoError(err)
			s.False(reflect.DeepEqual(hash, common.Hash{}))
		})
	}
}
