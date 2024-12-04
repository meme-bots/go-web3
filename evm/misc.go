package evm

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func sortAddressess(tkn0, tkn1 common.Address) (common.Address, common.Address) {
	token0Rep := new(big.Int).SetBytes(tkn0.Bytes())
	token1Rep := new(big.Int).SetBytes(tkn1.Bytes())
	if token0Rep.Cmp(token1Rep) > 0 {
		tkn0, tkn1 = tkn1, tkn0
	}
	return tkn0, tkn1
}

func CalculatePoolAddress(tokenA, tokenB, factoryAddr common.Address, poolInitCodeStr string) (poolAddr common.Address, err error) {
	poolInitCode, err := hex.DecodeString(poolInitCodeStr)
	if err != nil {
		return common.Address{}, err
	}

	tkn0, tkn1 := sortAddressess(tokenA, tokenB)
	msg := []byte{255}
	msg = append(msg, factoryAddr.Bytes()...)
	addrBytes := tkn0.Bytes()
	addrBytes = append(addrBytes, tkn1.Bytes()...)
	msg = append(msg, crypto.Keccak256(addrBytes)...)

	msg = append(msg, poolInitCode...)
	hash := crypto.Keccak256(msg)
	pairAddressBytes := big.NewInt(0).SetBytes(hash)
	pairAddressBytes = pairAddressBytes.Abs(pairAddressBytes)
	return common.BytesToAddress(pairAddressBytes.Bytes()), nil
}
