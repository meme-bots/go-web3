package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/meme-bots/go-web3/evm/multisend"
	t "github.com/meme-bots/go-web3/types"
	"github.com/samber/lo"
)

var (
	MULTISEND_ADDRESS = common.HexToAddress("0x5FcC77CE412131daEB7654b3D18ee89b13d86Cbf")
)

func ChainID(client *ethclient.Client) (uint64, error) {
	cid, err := client.ChainID(context.Background())
	if err != nil {
		return 0, err
	}
	return cid.Uint64(), nil
}

func TransferETH(client *ethclient.Client, chainID uint64, recipient common.Address, amount, gasPrice *big.Int, privateKey string) (common.Hash, error) {
	ctx := context.Background()
	fromPrivateKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Hash{}, err
	}
	fromAddr := crypto.PubkeyToAddress(fromPrivateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return common.Hash{}, err
	}

	tx := types.NewTx(&types.LegacyTx{Nonce: nonce, GasPrice: gasPrice, Gas: 21000, To: &recipient, Value: amount})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(new(big.Int).SetUint64(chainID)), fromPrivateKey)
	if err != nil {
		return common.Hash{}, err
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, err
	}

	return signedTx.Hash(), nil
}

func TransferETHBatch(
	client *ethclient.Client,
	chainID uint64,
	gasPrice *big.Int,
	privateKey string,
	bills []*t.TransferBill,
) (common.Hash, error) {
	fromPrivateKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Hash{}, err
	}
	fromAddr := crypto.PubkeyToAddress(fromPrivateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return common.Hash{}, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(fromPrivateKey, new(big.Int).SetUint64(chainID))
	if err != nil {
		return common.Hash{}, err
	}
	auth.Nonce = new(big.Int).SetUint64(nonce)
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(21000 * len(bills))
	auth.GasPrice = gasPrice

	ms, err := multisend.NewMultisend(MULTISEND_ADDRESS, client)
	if err != nil {
		return common.Hash{}, err
	}

	totalAmount := big.NewInt(0)
	for _, bill := range bills {
		totalAmount = new(big.Int).Add(totalAmount, bill.Amount)
	}

	auth.Value = totalAmount

	addresses := lo.Map(bills, func(b *t.TransferBill, _ int) common.Address { return common.HexToAddress(b.Recipient) })
	amounts := lo.Map(bills, func(b *t.TransferBill, _ int) *big.Int { return b.Amount })

	tx, err := ms.MultiTransfer(auth, addresses, amounts)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), nil
}
