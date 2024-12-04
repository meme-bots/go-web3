package utils

import (
	"math/big"

	"github.com/shopspring/decimal"
)

func CalculatePriceImpact(baseReserve, quoteReserve, solIncrement *big.Int) decimal.Decimal {
	K := new(big.Int).Mul(baseReserve, quoteReserve)
	newQuoteReserve := new(big.Int).Add(quoteReserve, solIncrement)
	newBaseReserve := new(big.Int).Div(K, newQuoteReserve)
	oldPrice := decimal.NewFromBigInt(quoteReserve, 0).Div(decimal.NewFromBigInt(baseReserve, 0))
	newPrice := decimal.NewFromBigInt(newQuoteReserve, 0).Div(decimal.NewFromBigInt(newBaseReserve, 0))
	return newPrice.Sub(oldPrice).Mul(decimal.NewFromInt(100)).Div(oldPrice)
}

func CalculateOutput(inputA, reserveA, reserveB uint64) uint64 {
	return decimal.NewFromUint64(inputA).Mul(decimal.NewFromUint64(reserveB)).
		Div(decimal.NewFromUint64(inputA + reserveA)).BigInt().Uint64()
}

func CalculateOutputBigInt(inputA, reserveA, reserveB *big.Int) *big.Int {
	return new(big.Int).Div(new(big.Int).Mul(inputA, reserveB), new(big.Int).Add(inputA, reserveA))
}
