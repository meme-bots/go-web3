package sol

import (
	"context"
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/types"
)

const MaxRecipientCount = 21

func SendTransfer(
	ctx context.Context,
	url string,
	recipient solana.PublicKey,
	amount uint64,
	privKey solana.PrivateKey,
	recentBlockHash solana.Hash,
) (solana.Signature, error) {
	instruction := system.NewTransferInstruction(amount, privKey.PublicKey(), recipient).Build()

	client := rpc.New(url)
	if recentBlockHash.IsZero() {
		latestBlock, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return solana.Signature{}, err
		}
		recentBlockHash = latestBlock.Value.Blockhash
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recentBlockHash,
		solana.TransactionPayer(privKey.PublicKey()),
	)
	if err != nil {
		return solana.Signature{}, err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		return &privKey
	})
	if err != nil {
		return solana.Signature{}, err
	}

	return client.SendTransaction(ctx, tx)
}

func SendTransferBatch(
	ctx context.Context,
	url string,
	privKey solana.PrivateKey,
	bills []*types.TransferBill,
	recentBlockHash solana.Hash,
) (solana.Signature, error) {

	if len(bills) > MaxRecipientCount {
		return solana.Signature{}, errors.New("exceeding the max recipients count")
	}

	instructions := make([]solana.Instruction, len(bills))
	for i, bill := range bills {
		instructions[i] = system.NewTransferInstruction(bill.Amount.Uint64(), privKey.PublicKey(), solana.MustPublicKeyFromBase58(bill.Recipient)).Build()
	}

	client := rpc.New(url)
	if recentBlockHash.IsZero() {
		latestBlock, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return solana.Signature{}, err
		}
		recentBlockHash = latestBlock.Value.Blockhash
	}

	tx, err := solana.NewTransaction(
		instructions,
		recentBlockHash,
		solana.TransactionPayer(privKey.PublicKey()),
	)
	if err != nil {
		return solana.Signature{}, err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		return &privKey
	})
	if err != nil {
		return solana.Signature{}, err
	}

	return client.SendTransaction(ctx, tx)
}
