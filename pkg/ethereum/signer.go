package ethereum

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

type Signer interface {
	Sign(payload []byte) ([]byte, error)
}

type PrivateKeySigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewPrivateKeySigner(privateKey *ecdsa.PrivateKey) *PrivateKeySigner {
	return &PrivateKeySigner{
		privateKey,
	}
}

func (s *PrivateKeySigner) Sign(payload []byte) ([]byte, error) {
	return crypto.Sign(payload, s.privateKey)
}
