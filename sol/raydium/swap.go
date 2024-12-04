package raydium

import (
	"context"
	"math/big"
	"math/rand"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/sol/common"
	"github.com/meme-bots/go-web3/utils"
)

func SendBuy(
	ctx context.Context,
	url string,
	marketID, marketProgramID, botFeeRecipient solana.PublicKey,
	solAmount, slippage, reserveWsol, reserveToken, gasFee, feeRatio, jitoTip uint64,
	market Market,
	createAta bool,
	privKey solana.PrivateKey,
	recentBlockHash solana.Hash,
) (solana.Signature, error) {
	var instructions []solana.Instruction
	//set gas limit and gas price
	gasLimit := 140000
	gasTmp := new(big.Int).Mul(big.NewInt(1000000), new(big.Int).SetUint64(gasFee))
	gasPrice := new(big.Int).Div(gasTmp, big.NewInt(int64(gasLimit)))
	setCULimitInst := computebudget.NewSetComputeUnitLimitInstruction(140000).Build()
	setCUPriceInst := computebudget.NewSetComputeUnitPriceInstruction(gasPrice.Uint64()).Build()
	instructions = append(instructions, setCULimitInst, setCUPriceInst)

	fee := solAmount * feeRatio / 10000
	solAmount -= fee

	//create and init tmp wsol token account
	createAndInitInsts, wsolAta := CreateAndInitWsolTokenAccount(privKey.PublicKey(), solAmount)
	instructions = append(instructions, createAndInitInsts...)
	//create ata
	var tokenMint solana.PublicKey
	if market.BaseMint.Equals(solana.SolMint) {
		tokenMint = market.QuoteMint
	} else {
		tokenMint = market.BaseMint
	}
	if createAta {
		createIdempotentInst := CreateIdempotentInstruction(tokenMint, privKey.PublicKey())
		instructions = append(instructions, createIdempotentInst)
	}
	//swap
	ata, _, _ := solana.FindAssociatedTokenAddress(privKey.PublicKey(), tokenMint)
	vaultSigner, _ := FindVaultSigner(market.VaultSignerNonce, marketID, marketProgramID)
	minAmountOut := utils.CalculateOutput(solAmount, reserveWsol, reserveToken)
	minAmountOut -= minAmountOut * slippage / 10000

	swapParam := CreateSwapParam{
		MarketId:      marketID,
		Maker:         privKey.PublicKey(),
		AmountIn:      solAmount,
		MinAmountOut:  minAmountOut,
		Market:        market,
		VaultSigner:   vaultSigner,
		SourceAccount: wsolAta,
		DestAccount:   ata,
	}
	swapInst := CreateSwapInstruction(swapParam)
	instructions = append(instructions, swapInst)
	//close wsol token account
	closeAccountInst := token.NewCloseAccountInstruction(
		wsolAta,
		privKey.PublicKey(),
		privKey.PublicKey(),
		[]solana.PublicKey{privKey.PublicKey()},
	).Build()
	instructions = append(instructions, closeAccountInst)
	//transfer fee
	if fee != 0 {
		feeTransferInst := system.NewTransferInstruction(fee, privKey.PublicKey(), botFeeRecipient).Build()
		instructions = append(instructions, feeTransferInst)
	}
	//jito tip
	if jitoTip != 0 {
		idx := rand.Intn(len(common.JitoTipPaymentAccounts))
		jitoTipTransferInst := system.NewTransferInstruction(jitoTip, privKey.PublicKey(), common.JitoTipPaymentAccounts[idx]).Build()
		instructions = append(instructions, jitoTipTransferInst)
	}
	cli := rpc.New(url)
	if recentBlockHash.IsZero() {
		recentBlock, err := cli.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return solana.Signature{}, err
		}
		recentBlockHash = recentBlock.Value.Blockhash
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
	if jitoTip != 0 {
		return rpc.New(common.JitoRpc).SendTransaction(ctx, tx)
	}

	return cli.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{SkipPreflight: true})
}

func SendSell(
	ctx context.Context,
	url string,
	marketID, marketProgramID, botFeeRecipient solana.PublicKey,
	tokenAmount, slippage, reserveWsol, reserveToken, gasFee, feeRatio, jitoTip uint64,
	market Market,
	isSellAll bool,
	privKey solana.PrivateKey,
	recentBlockHash solana.Hash,
) (solana.Signature, error) {
	var instructions []solana.Instruction

	//set gas limit and gas price
	gasLimit := 140000
	gasTmp := new(big.Int).Mul(big.NewInt(1000000), new(big.Int).SetUint64(gasFee))
	gasPrice := new(big.Int).Div(gasTmp, big.NewInt(int64(gasLimit)))
	setCULimitInst := computebudget.NewSetComputeUnitLimitInstruction(uint32(gasLimit)).Build()
	setCUPriceInst := computebudget.NewSetComputeUnitPriceInstruction(gasPrice.Uint64()).Build()
	instructions = append(instructions, setCULimitInst, setCUPriceInst)

	//create and init tmp wsol token account
	createAndInitInsts, wsolAta := CreateAndInitWsolTokenAccount(privKey.PublicKey(), 0)
	instructions = append(instructions, createAndInitInsts...)

	var tokenMint solana.PublicKey
	if market.BaseMint.Equals(solana.SolMint) {
		tokenMint = market.QuoteMint
	} else {
		tokenMint = market.BaseMint
	}

	//swap
	ata, _, _ := solana.FindAssociatedTokenAddress(privKey.PublicKey(), tokenMint)
	vaultSigner, _ := FindVaultSigner(market.VaultSignerNonce, marketID, marketProgramID)
	minAmountOut := utils.CalculateOutput(tokenAmount, reserveToken, reserveWsol)
	fee := minAmountOut * feeRatio / 10000
	minAmountOut -= minAmountOut * slippage / 10000

	swapParam := CreateSwapParam{
		MarketId:      marketID,
		Maker:         privKey.PublicKey(),
		AmountIn:      tokenAmount,
		MinAmountOut:  minAmountOut,
		Market:        market,
		VaultSigner:   vaultSigner,
		SourceAccount: ata,
		DestAccount:   wsolAta,
	}
	swapInst := CreateSwapInstruction(swapParam)
	instructions = append(instructions, swapInst)

	//close wsol token account
	closeAccountInst := token.NewCloseAccountInstruction(
		wsolAta,
		privKey.PublicKey(),
		privKey.PublicKey(),
		[]solana.PublicKey{privKey.PublicKey()},
	).Build()
	instructions = append(instructions, closeAccountInst)

	//close ata
	if isSellAll {
		closeAccountInst := token.NewCloseAccountInstruction(
			ata,
			privKey.PublicKey(),
			privKey.PublicKey(),
			[]solana.PublicKey{privKey.PublicKey()},
		).Build()
		instructions = append(instructions, closeAccountInst)
	}

	//transfer fee
	if fee != 0 {
		feeTransferInst := system.NewTransferInstruction(fee, privKey.PublicKey(), botFeeRecipient).Build()
		instructions = append(instructions, feeTransferInst)
	}

	//jito tip
	if jitoTip != 0 {
		idx := rand.Intn(len(common.JitoTipPaymentAccounts))
		jitoTipTransferInst := system.NewTransferInstruction(jitoTip, privKey.PublicKey(), common.JitoTipPaymentAccounts[idx]).Build()
		instructions = append(instructions, jitoTipTransferInst)
	}

	cli := rpc.New(url)
	if recentBlockHash.IsZero() {
		recentBlock, err := cli.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return solana.Signature{}, err
		}
		recentBlockHash = recentBlock.Value.Blockhash
	}

	tx, err := solana.NewTransaction(instructions, recentBlockHash, solana.TransactionPayer(privKey.PublicKey()))
	if err != nil {
		return solana.Signature{}, err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		return &privKey
	})
	if err != nil {
		return solana.Signature{}, err
	}

	if jitoTip != 0 {
		return rpc.New(common.JitoRpc).SendTransaction(ctx, tx)
	}

	return cli.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{SkipPreflight: true})
}

func CreateAndInitWsolTokenAccount(wallet solana.PublicKey, amount uint64) ([]solana.Instruction, solana.PublicKey) {
	randomKey, _ := solana.NewRandomPrivateKey()
	seed := randomKey.PublicKey().String()[0:32]
	wsolTokenAccount, _ := solana.CreateWithSeed(wallet, seed, solana.TokenProgramID)
	createWithSeedInst := system.NewCreateAccountWithSeedInstruction(
		wallet,
		seed,
		common.RentATA+amount,
		165,
		solana.TokenProgramID,
		wallet,
		wsolTokenAccount,
		wallet,
	).Build()

	initializeAccountInst := token.NewInitializeAccountInstruction(
		wsolTokenAccount,
		solana.SolMint,
		wallet,
		solana.SysVarRentPubkey,
	).Build()
	return []solana.Instruction{createWithSeedInst, initializeAccountInst}, wsolTokenAccount
}

func ParseSwapInstruction(tx *rpc.GetTransactionResult, instructionIndex, ataIndex uint16) (*big.Int, *big.Int) {
	solSwaped := big.NewInt(0)
	tokenSwaped := big.NewInt(0)
	for _, innerInstruction := range tx.Meta.InnerInstructions {
		if innerInstruction.Index == instructionIndex {
			innerInstructions := innerInstruction.Instructions
			var wsolAmount, tokenAmount uint64
			if innerInstructions[0].Accounts[0] == ataIndex {
				_ = bin.NewBorshDecoder(innerInstructions[0].Data[1:]).Decode(&tokenAmount)
				_ = bin.NewBorshDecoder(innerInstructions[1].Data[1:]).Decode(&wsolAmount)
				solSwaped = new(big.Int).SetUint64(wsolAmount)
				tokenSwaped = new(big.Int).Neg(new(big.Int).SetUint64(tokenAmount))
				return solSwaped, tokenSwaped
			} else if innerInstructions[1].Accounts[1] == ataIndex {
				_ = bin.NewBorshDecoder(innerInstructions[0].Data[1:]).Decode(&wsolAmount)
				_ = bin.NewBorshDecoder(innerInstructions[1].Data[1:]).Decode(&tokenAmount)
				solSwaped = new(big.Int).Neg(new(big.Int).SetUint64(wsolAmount))
				tokenSwaped = new(big.Int).SetUint64(tokenAmount)
				return solSwaped, tokenSwaped
			}
		}
	}
	return solSwaped, tokenSwaped
}
