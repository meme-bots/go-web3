package types

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type (
	Pool struct {
		ID              uint
		AmmPublicKey    string
		MarketPublicKey string
		MarketProgramID string
		BaseMint        string
		BaseVault       string
		BaseDecimal     uint8
		QuoteMint       string
		QuoteVault      string
		QuoteDecimal    uint8
		LpMint          string
		OpenTime        int64
		Status          int
		Dex             int
		Marked          bool
		MMType          RaydiumMMType
	}

	PriceHistorical struct {
		M5  decimal.Decimal
		H1  decimal.Decimal
		H6  decimal.Decimal
		H24 decimal.Decimal
	}

	Transact struct {
		ID               uint
		TelegramID       int64
		WalletID         int
		Owner            string
		TokenIn          string
		TokenInDecimals  int
		TokenOut         string
		TokenOutDecimals int
		PoolID           string
		MarketId         string
		MarketProgramId  string
		InAmount         *big.Int
		Gas              *big.Int
		Tip              *big.Int
		BotFee           *big.Int
		SlipPage         int
		Boot             bool
		TxHash           *string
		StateID          int
		OutAmount        *big.Int
		Dex              uint32
		TokenReserve     *big.Int
		QuoteReserve     *big.Int
		Allowance        *big.Int
	}
)
