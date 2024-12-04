package types

type (
	Config struct {
		Type                int
		Name                string
		RPC                 string
		WSRPC               string
		NativeTokenSymbol   string
		NativeTokenDecimals uint8

		QueryRPC       string // sol only
		WatchBlockHash bool

		Router                  string // evm only
		WrapNativeToken         string
		NativeTokenOracle       string
		UniswapPairInitCodeHash string
	}
)
