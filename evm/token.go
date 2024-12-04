package evm

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/meme-bots/go-web3/evm/erc20"
	"github.com/meme-bots/go-web3/evm/uniswap"
)

// type ChainConfig struct {
// 	chainId        uint64
// 	routerAddr     common.Address
// 	nativeAddr     common.Address
// 	nativeDecimals uint8
// }

// const (
// 	//Ethereum
// 	EthereumChainID                = 1
// 	EthereumUniswapRouterV2Address = "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
// 	EthereumWrapNativeAddress      = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
// 	EthereumWrapNativeDecimals     = 18

// 	//BSC
// 	BSCChainID                = 56
// 	BSCUniswapRouterV2Address = "0x10ED43C718714eb63d5aA57B78B54704E256024E"
// 	BSCWrapNativeAddress      = "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
// 	BSCWrapNativeDecimals     = 18

// 	//Base
// 	BaseChainID                = 8453
// 	BaseUniswapRouterV2Address = "0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24"
// 	BaseWrapNativeAddress      = "0x4200000000000000000000000000000000000006"
// 	BaseWrapNativeDecimals     = 18
// )

var (
	unlimitedApproveAmount, _ = new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	// chainMap = map[uint64]ChainConfig{
	// 	EthereumChainID: {
	// 		chainId:        EthereumChainID,
	// 		routerAddr:     common.HexToAddress(EthereumUniswapRouterV2Address),
	// 		nativeAddr:     common.HexToAddress(EthereumWrapNativeAddress),
	// 		nativeDecimals: EthereumWrapNativeDecimals,
	// 	},
	// 	BSCChainID: {
	// 		chainId:        BSCChainID,
	// 		routerAddr:     common.HexToAddress(BSCUniswapRouterV2Address),
	// 		nativeAddr:     common.HexToAddress(BSCWrapNativeAddress),
	// 		nativeDecimals: BSCWrapNativeDecimals,
	// 	},
	// 	BaseChainID: {
	// 		chainId:        BaseChainID,
	// 		routerAddr:     common.HexToAddress(BaseUniswapRouterV2Address),
	// 		nativeAddr:     common.HexToAddress(BaseWrapNativeAddress),
	// 		nativeDecimals: BaseWrapNativeDecimals,
	// 	},
	// }
)

func SwapBuy(
	cli *ethclient.Client,
	chainId uint64,
	routerAddr, wrappedAddr, tokenAddr common.Address,
	in, minOut, gasPrice *big.Int,
	privateKey string,
) (common.Hash, error) {
	router, err := uniswap.NewRouterv2(routerAddr, cli)
	if err != nil {
		return common.Hash{}, err
	}

	senderPriKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(senderPriKey, new(big.Int).SetUint64(chainId))
	if err != nil {
		return common.Hash{}, err
	}

	deadline := big.NewInt(time.Now().Unix() + 3600)
	senderAddress := crypto.PubkeyToAddress(senderPriKey.PublicKey)
	tx, err := router.SwapExactETHForTokens(
		&bind.TransactOpts{From: senderAddress, Signer: auth.Signer, Value: in, GasPrice: gasPrice},
		minOut,
		[]common.Address{wrappedAddr, tokenAddr},
		senderAddress,
		deadline,
	)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), err
}

func Allowerance(cli *ethclient.Client, spender common.Address, tokenAddr common.Address, owner common.Address) (*big.Int, error) {
	token, err := erc20.NewErc20(tokenAddr, cli)
	if err != nil {
		return nil, err
	}
	return token.Allowance(&bind.CallOpts{}, owner, spender)
}

func Approve(
	cli *ethclient.Client,
	chainId uint64,
	tokenAddr, spenderAddr common.Address,
	gasPrice *big.Int,
	privateKey string,
) (common.Hash, error) {
	token, err := erc20.NewErc20(tokenAddr, cli)
	if err != nil {
		return common.Hash{}, err
	}

	senderPriKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(senderPriKey, new(big.Int).SetUint64(chainId))
	if err != nil {
		return common.Hash{}, err
	}

	senderAddress := crypto.PubkeyToAddress(senderPriKey.PublicKey)
	tx, err := token.Approve(
		&bind.TransactOpts{From: senderAddress, Signer: auth.Signer, GasPrice: gasPrice},
		spenderAddr,
		unlimitedApproveAmount,
	)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), err
}

func SwapSell(
	cli *ethclient.Client,
	chainId uint64,
	routerAddr, tokenAddr, wrappedAddr common.Address,
	in, minOut, gasPrice *big.Int,
	privateKey string,
) (common.Hash, error) {
	router, err := uniswap.NewRouterv2(routerAddr, cli)
	if err != nil {
		return common.Hash{}, err
	}

	senderPriKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(senderPriKey, new(big.Int).SetUint64(chainId))
	if err != nil {
		return common.Hash{}, err
	}

	deadline := big.NewInt(time.Now().Unix() + 3600)
	senderAddress := crypto.PubkeyToAddress(senderPriKey.PublicKey)

	tx, err := router.SwapExactTokensForETH(
		&bind.TransactOpts{From: senderAddress, Signer: auth.Signer, GasPrice: gasPrice},
		in, minOut,
		[]common.Address{tokenAddr, wrappedAddr},
		senderAddress, deadline,
	)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), err
}
