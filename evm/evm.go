package evm

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	t "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/forta-network/go-multicall"
	mc "github.com/forta-network/go-multicall/contracts/contract_multicall"
	"github.com/meme-bots/go-web3/evm/erc20"
	"github.com/meme-bots/go-web3/evm/uniswap"
	"github.com/meme-bots/go-web3/types"
	"github.com/meme-bots/go-web3/utils"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type EVM struct {
	ctx     context.Context
	cfg     *types.Config
	client  *ethclient.Client
	chainId uint64
	watcher *Watcher
	factory common.Address
}

const (
	BUY_GAS  uint64 = 138263
	SELL_GAS uint64 = 138263
)

func NewEVM(
	ctx context.Context,
	cfg *types.Config,
) (*EVM, error) {
	client, err := ethclient.Dial(cfg.RPC)
	if err != nil {
		return nil, err
	}

	chainId, err := ChainID(client)
	if err != nil {
		return nil, err
	}

	router, err := uniswap.NewRouterv2(common.HexToAddress(cfg.Router), client)
	if err != nil {
		return nil, err
	}

	factory, err := router.Factory(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	watcher, err := NewWatcher(cfg.RPC, common.HexToAddress(cfg.NativeTokenOracle))
	if err != nil {
		return nil, err
	}

	return &EVM{
		ctx:     ctx,
		cfg:     cfg,
		client:  client,
		chainId: chainId,
		watcher: watcher,
		factory: factory,
	}, nil
}

func (s *EVM) Start() error {
	return s.watcher.Start()
}

func (s *EVM) Close() error {
	return s.watcher.Close()
}

func (v *EVM) GetType() int {
	return types.NetworkTypeEVM
}

func (v *EVM) GetTypeSymbol() string {
	return "EVM"
}

func (v *EVM) GetNativeTokenSymbol() string {
	return v.cfg.NativeTokenSymbol
}

func (v *EVM) GetNativeTokenDecimals() uint8 {
	return v.cfg.NativeTokenDecimals
}

func (v *EVM) GetNativeTokenPrice() decimal.Decimal {
	return v.watcher.GetETHPrice()
}

func (v *EVM) GetMaxMultiSendCount() int {
	return 100
}

func (v *EVM) GetGasPrice() *big.Int {
	return v.watcher.GetGasPrice()
}

func (v *EVM) GetBalance(req *types.GetBalanceRequest) (*big.Int, error) {
	return v.client.BalanceAt(v.ctx, common.HexToAddress(req.Address), nil)
}

func (v *EVM) GetTokenBalance(req *types.GetTokenBalanceRequest) (*big.Int, error) {
	token, err := erc20.NewErc20(common.HexToAddress(req.Token), v.client)
	if err != nil {
		return nil, err
	}
	return token.BalanceOf(&bind.CallOpts{}, common.HexToAddress(req.Owner))
}

func (v *EVM) GetPool(req *types.GetPoolRequest, pool *types.Pool) (*types.GetPoolResponse, error) {
	type balanceOutput struct {
		Balance *big.Int
	}

	type stringOutput struct {
		Value string
	}

	type uint8Output struct {
		Value uint8
	}

	type reservesOutput struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}

	poolAddr, err := CalculatePoolAddress(
		common.HexToAddress(v.cfg.WrapNativeToken),
		common.HexToAddress(req.Token),
		v.factory,
		v.cfg.UniswapPairInitCodeHash,
	)
	if err != nil {
		return nil, err
	}

	token0, _ := sortAddressess(common.HexToAddress(req.Token), common.HexToAddress(v.cfg.WrapNativeToken))
	isToken0 := req.Token == token0.String()

	caller, err := multicall.Dial(context.Background(), v.cfg.RPC)
	if err != nil {
		return nil, err
	}

	erc20Contract, err := multicall.NewContract(erc20.Erc20ABI, req.Token)
	if err != nil {
		return nil, err
	}

	routerContract, err := multicall.NewContract(uniswap.PairABI, poolAddr.String())
	if err != nil {
		return nil, err
	}

	mcContract, err := multicall.NewContract(mc.MulticallABI, multicall.DefaultAddress)
	if err != nil {
		return nil, err
	}

	calls, err := caller.Call(
		nil,
		erc20Contract.NewCall( // 0
			new(balanceOutput),
			"balanceOf",
			common.HexToAddress(req.Owner),
		),
		erc20Contract.NewCall( // 1
			new(balanceOutput),
			"totalSupply",
		),
		erc20Contract.NewCall( // 2
			new(stringOutput),
			"name",
		),
		erc20Contract.NewCall( // 3
			new(stringOutput),
			"symbol",
		),
		erc20Contract.NewCall( // 4
			new(uint8Output),
			"decimals",
		),
		erc20Contract.NewCall( // 5
			new(balanceOutput),
			"allowance",
			common.HexToAddress(req.Owner),
			common.HexToAddress(v.cfg.Router),
		),
		routerContract.NewCall( // 6
			new(reservesOutput),
			"getReserves",
		),
		mcContract.NewCall( // 7
			new(balanceOutput),
			"getEthBalance",
			common.HexToAddress(req.Owner),
		),
	)
	if err != nil {
		return nil, err
	}

	tokenBalance := calls[0].Outputs.(*balanceOutput).Balance
	totalSupplyWithDecimals := calls[1].Outputs.(*balanceOutput).Balance
	name := calls[2].Outputs.(*stringOutput).Value
	symbol := calls[3].Outputs.(*stringOutput).Value
	decimals := calls[4].Outputs.(*uint8Output).Value
	allowance := calls[5].Outputs.(*balanceOutput).Balance
	reserves := calls[6].Outputs.(*reservesOutput)
	nativeBalance := calls[7].Outputs.(*balanceOutput).Balance

	var tokenReserve, quoteReserve decimal.Decimal
	var tokenReserveBig, quoteReserveBig *big.Int
	if isToken0 {
		tokenReserve = decimal.NewFromBigInt(reserves.Reserve0, 0-int32(decimals))
		quoteReserve = decimal.NewFromBigInt(reserves.Reserve1, 0-int32(v.GetNativeTokenDecimals()))
		tokenReserveBig = reserves.Reserve0
		quoteReserveBig = reserves.Reserve1
	} else {
		tokenReserve = decimal.NewFromBigInt(reserves.Reserve1, 0-int32(decimals))
		quoteReserve = decimal.NewFromBigInt(reserves.Reserve0, 0-int32(v.GetNativeTokenDecimals()))
		tokenReserveBig = reserves.Reserve1
		quoteReserveBig = reserves.Reserve0
	}

	totalSupply := decimal.NewFromBigInt(totalSupplyWithDecimals, 0-int32(decimals))

	nativeTokenPrice := v.GetNativeTokenPrice()
	priceInUSD := quoteReserve.Mul(nativeTokenPrice).Div(tokenReserve)

	return &types.GetPoolResponse{
		NativeTokenPrice:      nativeTokenPrice,
		PriceInNativeToken:    quoteReserve.Div(tokenReserve),
		PriceInUSD:            priceInUSD,
		TotalSupply:           totalSupply,
		MarketCap:             totalSupply.Mul(priceInUSD),
		FreezeDisabled:        true,
		Burnt:                 true,
		MintAuthorityDisabled: true,
		TokenReserve:          tokenReserveBig,
		QuoteReserve:          quoteReserveBig,
		DexID:                 0,
		Name:                  name,
		Symbol:                symbol,
		Decimals:              decimals,
		QuoteDecimals:         v.GetNativeTokenDecimals(),
		TokenAddress:          req.Token,
		QuoteAddress:          v.cfg.WrapNativeToken,
		PoolAddress:           poolAddr.String(),
		NativeBalance:         nativeBalance,
		TokenBalance:          tokenBalance,
		Allowance:             allowance,
	}, nil
}

func (v *EVM) WaitReceipt(filter ethereum.FilterQuery, txHash common.Hash, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	sink := make(chan t.Log)
	sub, err := v.client.SubscribeFilterLogs(ctx, filter, sink)
	if err != nil {
		return err
	}

	for {
		select {
		case err := <-sub.Err():
			return err
		case vLog := <-sink:
			if vLog.TxHash.Hex() == txHash.Hex() {
				return nil
			}
		}
	}
}

func (v *EVM) WatchTransaction(req *types.WatchTransactionRequest) (interface{}, error) {
	var last time.Duration = 0
	step := time.Millisecond * 500

	ctx := context.Background()
	for {
		tx, pending, err := v.client.TransactionByHash(ctx, common.HexToHash(req.TxHash))
		if err != nil && !errors.Is(err, ethereum.NotFound) {
			return nil, err
		}
		if err == nil && !pending {
			return tx, nil
		}
		time.Sleep(step)
		last = last + step
		if last >= req.Duration {
			return nil, types.ErrTransactionInvalid
		}
	}
}

func (v *EVM) GetTransaction(req *types.GetTransactionRequest) (*types.GetTransactionResponse, error) {
	tx, _ := req.Tx.(*t.Transaction)

	receipt, err := v.client.TransactionReceipt(v.ctx, common.HexToHash(req.TxHash))
	if err != nil {
		return nil, err
	}

	transferTopic := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	fee := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), receipt.EffectiveGasPrice)

	var balanceChanged, tokenChanged *big.Int
	{
		tokenLog, ok := lo.Find(receipt.Logs, func(n *t.Log) bool {
			return n.Address.String() == req.Token && n.Topics[0] == transferTopic
		})
		if !ok {
			return nil, types.ErrTransactionInvalid
		}

		token, err := erc20.NewErc20(common.HexToAddress(req.Token), nil)
		if err != nil {
			return nil, err
		}
		tokenTransfer, err := token.ParseTransfer(*tokenLog)
		if err != nil {
			return nil, err
		}
		tokenChanged = tokenTransfer.Value
		if tokenTransfer.From.String() == req.Owner {
			tokenChanged = new(big.Int).Neg(tokenChanged)
		}
	}

	var botFee *big.Int
	value := tx.Value()
	{
		wethLog, ok := lo.Find(receipt.Logs, func(n *t.Log) bool {
			return n.Address.String() == v.cfg.WrapNativeToken && n.Topics[0] == transferTopic
		})
		if !ok {
			return nil, types.ErrTransactionInvalid
		}
		weth, err := erc20.NewErc20(common.HexToAddress(v.cfg.WrapNativeToken), nil)
		if err != nil {
			return nil, err
		}
		wethTransfer, err := weth.ParseTransfer(*wethLog)
		if err != nil {
			return nil, err
		}
		if value != nil && value.Cmp(big.NewInt(0)) > 0 {
			balanceChanged = new(big.Int).Neg(value)
			botFee = new(big.Int).Sub(value, wethTransfer.Value)
		} else {
			botFee = new(big.Int).Div(wethTransfer.Value, big.NewInt(100))
			balanceChanged = new(big.Int).Sub(wethTransfer.Value, botFee)
		}
	}

	return &types.GetTransactionResponse{
		BalanceChanged: balanceChanged,
		TokenChanged:   tokenChanged,
		Fee:            fee,
		BotFee:         botFee,
	}, nil
}

func (v *EVM) CheckAddress(text string) bool {
	return common.IsHexAddress(text)
}

func (v *EVM) CheckNormalizedAddress(text string) bool {
	return common.IsHexAddress(text)
}

func (v *EVM) GetAddressFromInput(text string) string {
	if v.CheckAddress(text) {
		return text
	}
	return ""
}

func (v *EVM) Withdraw(to string, amount decimal.Decimal, privateKey string) (string, error) {
	txHash, err := TransferETH(
		v.client,
		v.chainId,
		common.HexToAddress(to),
		amount.Mul(decimal.New(1, int32(v.cfg.NativeTokenDecimals))).BigInt(),
		v.GetGasPrice(),
		privateKey,
	)
	if err != nil {
		return "", err
	}
	return txHash.Hex(), nil
}

func (v *EVM) Launch(req *types.LaunchRequest, feeRecipient_ string, feeRatio uint64, privateKey string) (*types.LaunchResponse, error) {
	return nil, types.ErrNotImplemented
}

func (v *EVM) Transact(req *types.Transact, feeRecipient string, feeRatio uint64, privateKey string) (*types.TransactResponse, error) {
	buy := req.TokenIn == v.cfg.WrapNativeToken
	token := req.TokenIn
	if buy {
		token = req.TokenOut
	}

	var txHash common.Hash
	var err error
	var positionClosed = false
	initialTokenBalance, err := v.GetTokenBalance(&types.GetTokenBalanceRequest{Owner: req.Owner, Token: token})
	if err != nil {
		return nil, err
	}

	gasPrice := new(big.Int).Add(req.Gas, req.Tip)

	if buy {
		inAmount := new(big.Int).Div(new(big.Int).Mul(req.InAmount, big.NewInt(99)), big.NewInt(100))
		minAmountOut := utils.CalculateOutputBigInt(inAmount, req.QuoteReserve, req.TokenReserve)
		slip := new(big.Int).Div(new(big.Int).Mul(minAmountOut, big.NewInt(int64(req.SlipPage))), big.NewInt(10000))
		minAmountOut = new(big.Int).Sub(minAmountOut, slip)

		txHash, err = SwapBuy(
			v.client,
			v.chainId,
			common.HexToAddress(v.cfg.Router),
			common.HexToAddress(v.cfg.WrapNativeToken),
			common.HexToAddress(req.TokenOut),
			req.InAmount,
			minAmountOut,
			gasPrice,
			privateKey,
		)
	} else {
		if req.InAmount.Cmp(req.Allowance) > 0 {
			tx, err := Approve(
				v.client,
				v.chainId,
				common.HexToAddress(req.TokenIn),
				common.HexToAddress(v.cfg.Router),
				v.GetGasPrice(),
				privateKey,
			)
			if err != nil {
				return nil, err
			}
			_, err = v.WatchTransaction(&types.WatchTransactionRequest{TxHash: tx.String(), Duration: 30 * time.Second})
			if err != nil {
				return nil, err
			}
		}

		minAmountOut := utils.CalculateOutputBigInt(req.InAmount, req.TokenReserve, req.QuoteReserve)
		slip := new(big.Int).Div(new(big.Int).Mul(minAmountOut, big.NewInt(int64(req.SlipPage))), big.NewInt(10000))
		minAmountOut = new(big.Int).Sub(minAmountOut, slip)

		if req.InAmount.Cmp(initialTokenBalance) == 0 {
			positionClosed = true
		}

		txHash, err = SwapSell(
			v.client,
			v.chainId,
			common.HexToAddress(v.cfg.Router),
			common.HexToAddress(req.TokenIn),
			common.HexToAddress(v.cfg.WrapNativeToken),
			req.InAmount,
			minAmountOut,
			gasPrice,
			privateKey,
		)
	}

	if err != nil {
		return nil, err
	}

	return &types.TransactResponse{
		TxHash:              txHash.String(),
		InitialTokenBalance: initialTokenBalance,
		PositionClosed:      positionClosed,
	}, nil
}

func (v *EVM) GetBaseGas() *big.Int {
	return new(big.Int).Mul(v.GetGasPrice(), big.NewInt(21000))
}

func (v *EVM) SendNative(bill *types.TransferBill, privateKey string) (string, error) {
	txHash, err := TransferETH(
		v.client,
		v.chainId,
		common.HexToAddress(bill.Recipient),
		bill.Amount,
		v.GetGasPrice(),
		privateKey,
	)
	if err != nil {
		return "", err
	}
	return txHash.String(), nil
}

func (v *EVM) SendNativeBatch(bills []*types.TransferBill, privateKey string) (string, error) {
	txHash, err := TransferETHBatch(
		v.client,
		v.chainId,
		v.GetGasPrice(),
		privateKey,
		bills,
	)
	if err != nil {
		return "", err
	}
	return txHash.String(), nil
}
