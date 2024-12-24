package types

import (
	"math/big"
	"time"

	"github.com/shopspring/decimal"
)

type (
	GetBalanceRequest struct {
		Address string
	}

	GetTokenBalanceRequest struct {
		Owner string
		Token string
	}

	GetTokenRequest struct {
		Address string
	}

	GetTokenResponse struct {
		Symbol          string
		Name            string
		Address         string
		Decimals        int
		TotalSupply     uint64
		Price           float64
		PriceImpact     float64
		PoolAddress     string
		QuoteAddress    string
		TokenReserve    decimal.Decimal
		SolReserve      decimal.Decimal
		LaunchPad       string
		LaunchPadStatus int
	}

	QueryPoolRequest struct {
		Token string
	}

	QueryPoolResponse struct {
		PoolAddress string
	}

	LaunchRequest struct {
		DexID        int
		Name         string
		Symbol       string
		Description  string
		Uri          string
		Gas          *big.Int
		Tip          *big.Int
		BuyAmountSol *big.Int
		SlipPage     uint64
	}

	LaunchResponse struct {
		TxHash string
		Token  string
	}

	TransactResponse struct {
		TxHash              string
		InitialTokenBalance *big.Int
		PositionClosed      bool
	}

	WatchTransactionRequest struct {
		TxHash   string
		Duration time.Duration
	}

	GetTransactionRequest struct {
		TxHash       string
		Owner        string
		Token        string
		Tx           interface{}
		FeeRecipient string
	}

	GetTransactionResponse struct {
		BalanceChanged *big.Int
		TokenChanged   *big.Int
		Fee            *big.Int
		BotFee         *big.Int
	}

	GetPoolRequest struct {
		Token string
		URL   string
		Owner string
	}

	GetPoolResponse struct {
		NativeTokenPrice      decimal.Decimal
		PriceInNativeToken    decimal.Decimal
		PriceInUSD            decimal.Decimal
		TotalSupply           decimal.Decimal
		MarketCap             decimal.Decimal
		FreezeDisabled        bool
		Burnt                 bool
		MintAuthorityDisabled bool
		TokenReserve          *big.Int
		QuoteReserve          *big.Int
		DexID                 int
		Name                  string
		Symbol                string
		Decimals              uint8
		QuoteDecimals         uint8
		TokenAddress          string
		QuoteAddress          string
		PoolAddress           string
		MarketId              string
		MarketProgramId       string
		NativeBalance         *big.Int
		TokenBalance          *big.Int
		Allowance             *big.Int
	}

	TransferBill struct {
		Recipient string
		Amount    *big.Int
	}

	NetworkInterface interface {
		Start() error
		Close() error
		GetType() int
		GetTypeSymbol() string
		GetNativeTokenSymbol() string
		GetNativeTokenDecimals() uint8
		GetNativeTokenPrice() decimal.Decimal
		GetMaxMultiSendCount() int
		GetBalance(req *GetBalanceRequest) (*big.Int, error)
		GetTokenBalance(req *GetTokenBalanceRequest) (*big.Int, error)
		GetPool(token *GetPoolRequest) (*GetPoolResponse, error)
		WatchTransaction(req *WatchTransactionRequest) (interface{}, error)
		GetTransaction(req *GetTransactionRequest) (*GetTransactionResponse, error)
		CheckAddress(text string) bool
		CheckNormalizedAddress(text string) bool
		GetAddressFromInput(text string) string
		Withdraw(to string, amount decimal.Decimal, privateKey string) (string, error)
		Transact(req *Transact, feeRecipient_ string, feeRatio uint64, privateKey string) (*TransactResponse, error)
		GetBaseGas() *big.Int
		SendNative(bill *TransferBill, privateKey string) (string, error)
		SendNativeBatch(bills []*TransferBill, privateKey string) (string, error)
		Launch(req *LaunchRequest, feeRecipient_ string, feeRatio uint64, privateKey string) (*LaunchResponse, error)
	}
)

const (
	NetworkTypeSol int = iota
	NetworkTypeEVM
	NetworkTypeSUI
	NetworkTypeTON
)
