package pumpfun

import (
	"context"
	"math/big"
	"math/rand"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/sol/common"
	"github.com/meme-bots/go-web3/types"
	"github.com/meme-bots/go-web3/utils"
	"github.com/near/borsh-go"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

var (
	GlobalPubKey                  = solana.MPK("4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf")
	GlobalFeeRecipient            = solana.MPK("CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM")
	eventAuthority                = solana.MPK("Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1")
	MPL_TOKEN_METADATA_PROGRAM_ID = solana.MPK("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")
	MINT_AUTHORITY                = solana.MPK("TSLvdd1pWpHVjahSpsvCXUbgwsL3JAcvokwaKt1eokM")
)

const (
	TotalSupplyWithDecimals = 1000000000000000
	TotalSupply             = 1000000000

	// MINT_AUTHORITY_SEED = "mint-authority"
	// BONDING_CURVE_SEED  = "bonding-curve"
)

type SwapLogLayout struct {
	MethodID             uint64           `json:"methodId"`
	Mint                 solana.PublicKey `json:"mint"`
	SolAmount            uint64           `json:"solAmount"`
	TokenAmount          uint64           `json:"tokenAmount"`
	IsBuy                bool             `json:"isBuy"`
	User                 solana.PublicKey `json:"user"`
	Timestamp            int64            `json:"timestamp"`
	VirtualSolReserves   uint64           `json:"virtualSolReserves"`
	VirtualTokenReserves uint64           `json:"virtualTokenReserves"`
}

func GetPumpFunPool(ctx context.Context, url string, req *common.GetSolPoolRequest) (*common.GetSolPoolResponse, *common.Balance, error) {
	mint := solana.MPK(req.Token)
	client := rpc.New(url)

	bondingCurvePubKey := FindBondingCurve(mint)
	metaAddress, _, _ := solana.FindTokenMetadataAddress(mint)
	publicKeys := []solana.PublicKey{
		bondingCurvePubKey,
		mint,
		metaAddress,
	}

	if req.WithBalance {
		owner := solana.MPK(req.Owner)
		ata, _, _ := solana.FindAssociatedTokenAddress(owner, mint)
		publicKeys = append(publicKeys, owner, ata)
	}

	accounts, err := client.GetMultipleAccountsWithOpts(
		ctx,
		publicKeys,
		&rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentProcessed},
	)
	if err != nil {
		return nil, nil, err
	}

	var balance *common.Balance = nil
	if req.WithBalance {
		balance = &common.Balance{NativeBalance: big.NewInt(0), TokenBalance: big.NewInt(0)}

		if accounts.Value[3] != nil {
			balance.NativeBalance = new(big.Int).SetUint64(accounts.Value[3].Lamports)
		}

		if accounts.Value[4] != nil {
			var tokenAccount token.Account
			err = tokenAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[4].Data.GetBinary()))
			if err != nil {
				return nil, nil, err
			}
			balance.TokenBalance = new(big.Int).SetUint64(tokenAccount.Amount)
		}
	}

	if accounts.Value[0] == nil {
		return nil, balance, types.ErrInvalidPool
	}

	bondingCurve := BondingCurve{}
	err = borsh.Deserialize(&bondingCurve, accounts.Value[0].Data.GetBinary())
	if err != nil {
		return nil, balance, err
	}

	if bondingCurve.Complete || bondingCurve.VirtualTokenReserves == 0 {
		return nil, balance, types.ErrPoolCompleted
	}

	var tokenMintAccount token.Mint
	err = tokenMintAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[1].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}

	meta, err := common.MetadataDeserialize(accounts.Value[2].Data.GetBinary())
	if err != nil {
		return nil, balance, err
	}

	solAmount := decimal.NewFromUint64(bondingCurve.VirtualSolReserves).Div(decimal.NewFromInt(1e9))
	tokenAmount := decimal.NewFromUint64(bondingCurve.VirtualTokenReserves).Div(decimal.NewFromInt(1e6))
	priceInSol := solAmount.Div(tokenAmount)

	return &common.GetSolPoolResponse{
		PriceInSol:            priceInSol,
		TotalSupply:           decimal.NewFromUint64(TotalSupply),
		FreezeDisabled:        true,
		Burnt:                 true,
		MintAuthorityDisabled: true,
		TokenReserve:          new(big.Int).SetUint64(bondingCurve.VirtualTokenReserves),
		SolReserve:            new(big.Int).SetUint64(bondingCurve.VirtualSolReserves),
		Name:                  utils.TrimSpace(meta.Data.Name),
		Symbol:                utils.TrimSpace(meta.Data.Symbol),
		Decimals:              tokenMintAccount.Decimals,
		QuoteDecimals:         9,
		TokenAddress:          req.Token,
		QuoteAddress:          solana.SolMint.String(),
		PoolAddress:           bondingCurvePubKey.String(),
	}, balance, nil
}

func GetPumpFunPoolByToken(ctx context.Context, url string, token solana.PublicKey) (*types.Pool, error) {
	client := rpc.New(url)

	bondingCurvePubKey := FindBondingCurve(token)
	baseVault, _, _ := solana.FindAssociatedTokenAddress(bondingCurvePubKey, token)
	accountInfo, err := client.GetAccountInfoWithOpts(ctx, bondingCurvePubKey, &rpc.GetAccountInfoOpts{Commitment: rpc.CommitmentConfirmed})
	if err != nil {
		return nil, err
	}

	bondingCurve := BondingCurve{}
	err = borsh.Deserialize(&bondingCurve, accountInfo.GetBinary())
	if err != nil {
		return nil, err
	}

	return &types.Pool{
		AmmPublicKey: bondingCurvePubKey.String(),
		BaseMint:     token.String(),
		BaseVault:    baseVault.String(),
		BaseDecimal:  6,
		QuoteMint:    solana.SolMint.String(),
		QuoteVault:   bondingCurvePubKey.String(),
		QuoteDecimal: 9,
		Status:       lo.If(bondingCurve.Complete, 1).Else(0),
		Dex:          1,
		Marked:       true,
	}, nil
}

func FindBondingCurve(mint solana.PublicKey) solana.PublicKey {
	bondingCurve, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("bonding-curve"),
			mint.Bytes(),
		},
		ProgramID,
	)
	return bondingCurve
}

func SendBuy(
	ctx context.Context,
	url string,
	mint, botFeeRecipient solana.PublicKey,
	solAmount, slippage, gasFee, feeRatio, jitoTip uint64,
	bondingCurve BondingCurve,
	createAta bool,
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

	bondingCurvePubKey := FindBondingCurve(mint)
	bondingCurveAta, _, _ := solana.FindAssociatedTokenAddress(bondingCurvePubKey, mint)
	ata, _, _ := solana.FindAssociatedTokenAddress(privKey.PublicKey(), mint)

	//create ata
	if createAta {
		createAtaInst := associatedtokenaccount.NewCreateInstruction(privKey.PublicKey(), privKey.PublicKey(), mint).Build()
		instructions = append(instructions, createAtaInst)
	}

	fee := solAmount * feeRatio / 10000
	solAmount -= fee
	tokenAmount := utils.CalculateOutput(solAmount, bondingCurve.VirtualSolReserves, bondingCurve.VirtualTokenReserves)
	solAmount += solAmount * slippage / 10000

	//swap
	swapInst := NewBuyInstruction(
		tokenAmount,
		solAmount,
		GlobalPubKey,
		GlobalFeeRecipient,
		mint,
		bondingCurvePubKey,
		bondingCurveAta,
		ata,
		privKey.PublicKey(),
		solana.SystemProgramID,
		solana.TokenProgramID,
		solana.SysVarRentPubkey,
		eventAuthority,
		ProgramID,
	).Build()
	instructions = append(instructions, swapInst)

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

func SendSell(
	ctx context.Context,
	url string,
	mint, botFeeRecipient solana.PublicKey,
	tokenAmount, slippage, gasFee, feeRatio, jitoTip uint64,
	bondingCurve BondingCurve,
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

	bondingCurvePubKey := FindBondingCurve(mint)
	bondingCurveAta, _, _ := solana.FindAssociatedTokenAddress(bondingCurvePubKey, mint)
	ata, _, _ := solana.FindAssociatedTokenAddress(privKey.PublicKey(), mint)
	minSolOutput := utils.CalculateOutput(tokenAmount, bondingCurve.VirtualTokenReserves, bondingCurve.VirtualSolReserves)
	fee := minSolOutput * feeRatio / 10000
	minSolOutput -= minSolOutput * slippage / 10000
	//swap
	swapInst := NewSellInstruction(
		tokenAmount,
		minSolOutput,
		GlobalPubKey,
		GlobalFeeRecipient,
		mint,
		bondingCurvePubKey,
		bondingCurveAta,
		ata,
		privKey.PublicKey(),
		solana.SystemProgramID,
		solana.SPLAssociatedTokenAccountProgramID,
		solana.TokenProgramID,
		eventAuthority,
		ProgramID,
	).Build()
	instructions = append(instructions, swapInst)

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

func GetInitialBuyPrice(global *Global, solAmount uint64) uint64 {
	if solAmount == 0 {
		return 0
	}

	k := new(big.Int).Mul(new(big.Int).SetUint64(global.InitialVirtualSolReserves), new(big.Int).SetUint64(global.InitialVirtualTokenReserves))
	i := new(big.Int).SetUint64(global.InitialVirtualSolReserves + solAmount)
	r := new(big.Int).Add(new(big.Int).Div(k, i), big.NewInt(1))
	s := new(big.Int).Sub(new(big.Int).SetUint64(global.InitialVirtualTokenReserves), r)
	max := new(big.Int).SetUint64(global.InitialRealTokenReserves)
	if s.Cmp(max) < 0 {
		return s.Uint64()
	} else {
		return max.Uint64()
	}
}

func CreateAndBuy(
	ctx context.Context,
	rpcUrl, name, symbol, uri string,
	botFeeRecipient solana.PublicKey,
	solAmount, slippage, priorityFee, feeRatio, jitoTip uint64,
	global *Global,
	privKey solana.PrivateKey,
	recentBlockHash solana.Hash,
) (solana.Signature, solana.PublicKey, error) {
	var instructions []solana.Instruction

	if priorityFee > 0 {
		gasLimit := lo.If(solAmount > 0, 240000).Else(200000)
		gasTmp := new(big.Int).Mul(big.NewInt(1000000), new(big.Int).SetUint64(priorityFee))
		gasPrice := new(big.Int).Div(gasTmp, big.NewInt(int64(gasLimit)))
		setCULimitInst := computebudget.NewSetComputeUnitLimitInstruction(uint32(gasLimit)).Build()
		setCUPriceInst := computebudget.NewSetComputeUnitPriceInstruction(gasPrice.Uint64()).Build()
		instructions = append(instructions, setCULimitInst, setCUPriceInst)
	}

	owner := privKey.PublicKey()
	mintPrivateKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return solana.Signature{}, solana.PublicKey{}, err
	}

	mint := mintPrivateKey.PublicKey()
	metadataPDA, _, _ := solana.FindTokenMetadataAddress(mint)
	bondingCurvePubKey := FindBondingCurve(mint)
	bondingCurveAta, _, _ := solana.FindAssociatedTokenAddress(bondingCurvePubKey, mint)

	fee := solAmount * feeRatio / 10000
	solAmount -= fee

	createInst := NewCreateInstruction(
		name,
		symbol,
		uri,
		mint,
		MINT_AUTHORITY,
		bondingCurvePubKey,
		bondingCurveAta,
		GlobalPubKey,
		solana.TokenMetadataProgramID,
		metadataPDA,
		owner,
		solana.SystemProgramID,
		solana.TokenProgramID,
		solana.SPLAssociatedTokenAccountProgramID,
		solana.SysVarRentPubkey,
		eventAuthority,
		ProgramID,
	).Build()

	instructions = append(instructions, createInst)

	if solAmount > 0 {
		ata, _, _ := solana.FindAssociatedTokenAddress(owner, mint)
		createAtaInst := associatedtokenaccount.NewCreateInstruction(owner, owner, mint).Build()
		instructions = append(instructions, createAtaInst)

		tokenAmount := GetInitialBuyPrice(global, solAmount)
		solAmountWithSlippage := solAmount + (solAmount*slippage)/10000
		swapInst := NewBuyInstruction(
			tokenAmount,
			solAmountWithSlippage,
			GlobalPubKey,
			GlobalFeeRecipient,
			mint,
			bondingCurvePubKey,
			bondingCurveAta,
			ata,
			owner,
			solana.SystemProgramID,
			solana.TokenProgramID,
			solana.SysVarRentPubkey,
			eventAuthority,
			ProgramID,
		).Build()
		instructions = append(instructions, swapInst)
	}

	if fee != 0 {
		feeTransferInst := system.NewTransferInstruction(fee, owner, botFeeRecipient).Build()
		instructions = append(instructions, feeTransferInst)
	}

	if jitoTip != 0 {
		idx := rand.Intn(len(common.JitoTipPaymentAccounts))
		jitoTipTransferInst := system.NewTransferInstruction(jitoTip, owner, common.JitoTipPaymentAccounts[idx]).Build()
		instructions = append(instructions, jitoTipTransferInst)
	}

	cli := rpc.New(rpcUrl)
	if recentBlockHash.IsZero() {
		recentBlock, err := cli.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return solana.Signature{}, solana.PublicKey{}, err
		}
		recentBlockHash = recentBlock.Value.Blockhash
	}

	tx, err := solana.NewTransaction(instructions, recentBlockHash, solana.TransactionPayer(owner))
	if err != nil {
		return solana.Signature{}, solana.PublicKey{}, err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(owner) {
			return &privKey
		} else if key.Equals(mint) {
			return &mintPrivateKey
		}
		return nil
	})
	if err != nil {
		return solana.Signature{}, solana.PublicKey{}, err
	}

	var signature solana.Signature
	if jitoTip != 0 {
		signature, err = rpc.New(common.JitoRpc).SendTransaction(ctx, tx)
	} else {
		signature, err = cli.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{SkipPreflight: false})
	}

	if err != nil {
		return solana.Signature{}, solana.PublicKey{}, err
	}
	return signature, mint, nil
}
