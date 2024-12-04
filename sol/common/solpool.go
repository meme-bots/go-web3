package common

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type (
	GetSolPoolRequest struct {
		Owner       string
		Token       string
		Pool        string
		WithBalance bool
	}

	GetSolPoolResponse struct {
		PriceInSol            decimal.Decimal
		TotalSupply           decimal.Decimal
		FreezeDisabled        bool
		Burnt                 bool
		MintAuthorityDisabled bool
		TokenReserve          *big.Int
		SolReserve            *big.Int
		Name                  string
		Symbol                string
		Decimals              uint8
		QuoteDecimals         uint8
		TokenAddress          string
		QuoteAddress          string
		PoolAddress           string
		MarketId              string
		MarketProgramId       string
	}

	Balance struct {
		NativeBalance *big.Int
		TokenBalance  *big.Int
	}
)
