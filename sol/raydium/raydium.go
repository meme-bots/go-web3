package raydium

import (
	"context"
	"encoding/json"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/sol/common"
	"github.com/meme-bots/go-web3/types"
	"github.com/meme-bots/go-web3/utils"
	"github.com/near/borsh-go"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"math/big"
	"net/http"
)

const (
	InstructionSwap = 9

	RaydiumApiV3 = "https://api-v3.raydium.io"
)

var (
	CLMMProgramID = solana.MPK("CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK")
	ProgramID     = solana.MPK("675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8")
	AuthorityV4   = solana.MPK("5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1")
)

type (
	Instruction uint8

	Pool struct {
		Status                 uint64
		Nonce                  uint64
		MaxOrder               uint64
		Depth                  uint64
		BaseDecimal            uint64
		QuoteDecimal           uint64
		State                  uint64
		ResetFlag              uint64
		MinSize                uint64
		VolMaxCutRatio         uint64
		AmountWaveRatio        uint64
		BaseLotSize            uint64
		QuoteLotSize           uint64
		MinPriceMultiplier     uint64
		MaxPriceMultiplier     uint64
		SystemDecimalValue     uint64
		MinSeparateNumerator   uint64
		MinSeparateDenominator uint64
		TradeFeeNumerator      uint64
		TradeFeeDenominator    uint64
		PnlNumerator           uint64
		PnlDenominator         uint64
		SwapFeeNumerator       uint64
		SwapFeeDenominator     uint64
		SaseNeedTakePnl        uint64
		QuoteNeedTakePnl       uint64
		QuoteTotalPnl          uint64
		BaseTotalPnl           uint64
		PoolOpenTime           uint64
		PunishPcAmount         uint64
		PunishCoinAmount       uint64
		OrderbookToInitTime    uint64

		SwapBaseInAmount1   uint64
		SwapBaseInAmount2   uint64
		SwapQuoteOutAmount1 uint64
		SwapQuoteOutAmount2 uint64

		SwapBase2QuoteFee  uint64
		SwapQuoteInAmount1 uint64
		SwapQuoteInAmount2 uint64
		SwapBaseOutAmount1 uint64
		SwapBaseOutAmount2 uint64
		SwapQuote2BaseFee  uint64
		// amm vault
		BaseVault  solana.PublicKey
		QuoteVault solana.PublicKey
		// mint
		BaseMint  solana.PublicKey
		QuoteMint solana.PublicKey
		LpMint    solana.PublicKey
		// Market
		OpenOrders      solana.PublicKey
		MarketId        solana.PublicKey
		MarketProgramId solana.PublicKey
		TargetOrders    solana.PublicKey
		WithdrawQueue   solana.PublicKey
		LpVault         solana.PublicKey
		Owner           solana.PublicKey
		// true circulating supply without lock up
		LpReserve uint64

		Padding uint64
		// seq(u64(), 3, 'padding')
	}

	PoolInfo struct {
		ID              uint `json:"id"`
		AmmPublicKey    string
		MarketPublicKey string
		BaseMint        string
		QuoteMint       string
		QuoteVault      string
		BaseVault       string
		LpMint          string
		OpenTime        int64
		Marked          bool
		BaseDecimal     uint64
		QuoteDecimal    uint64
		MarketProgramId string
		Dex             int
		Status          int
	}

	RewardInfo struct {
		RewardState           uint8
		OpenTime              uint64
		EndTime               uint64
		LastUpdateTime        uint64
		EmissionsPerSecondX64 big.Int
		RewardTotalEmissioned uint64
		RewardClaimed         uint64
		TokenMint             solana.PublicKey
		TokenVault            solana.PublicKey
		Authority             solana.PublicKey
		RewardGrowthGlobalX64 big.Int
	}

	ClmmPoolData struct {
		Discrimator            [8]uint8
		Bump                   uint8
		AmmConfig              solana.PublicKey
		Owner                  solana.PublicKey
		TokenMint0             solana.PublicKey
		TokenMint1             solana.PublicKey
		TokenVault0            solana.PublicKey
		TokenVault1            solana.PublicKey
		ObservationKey         solana.PublicKey
		MintDecimals0          uint8
		MintDecimals1          uint8
		TickSpacing            uint16
		Liquidity              big.Int
		SqrtPriceX64           big.Int
		TickCurrent            int32
		Padding3               uint16
		Padding4               uint16
		FeeGrowthGlobal0X64    big.Int
		FeeGrowthGlobal1X64    big.Int
		ProtocolFeesToken0     uint64
		ProtocolFeesToken1     uint64
		SwapInAmountToken0     big.Int
		SwapOutAmountToken1    big.Int
		SwapInAmountToken1     big.Int
		SwapOutAmountToken0    big.Int
		Status                 uint8
		Padding                [7]uint8
		RewardInfos            [3]RewardInfo
		TickArrayBitmap        [16]uint64
		TotalFeesToken0        uint64
		TotalFeesClaimedToken0 uint64
		TotalFeesToken1        uint64
		TotalFeesClaimedToken1 uint64
		FundFeesToken0         uint64
		FundFeesToken1         uint64
		OpenTime               uint64
		RecentEpoch            uint64
		Padding1               [24]uint64
		Padding2               [32]uint64
	}
)

func GeRaydiumPoolP2(ctx context.Context, url string, p *types.Pool, owner string, withBalance bool) (*common.GetSolPoolResponse, *common.Balance, error) {
	baseMint, err := solana.PublicKeyFromBase58(p.BaseMint)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	quoteMint, err := solana.PublicKeyFromBase58(p.QuoteMint)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	baseVault, err := solana.PublicKeyFromBase58(p.BaseVault)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	quoteVault, err := solana.PublicKeyFromBase58(p.QuoteVault)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	lpMint, err := solana.PublicKeyFromBase58(p.LpMint)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	if !baseMint.Equals(solana.SolMint) && !quoteMint.Equals(solana.SolMint) {
		return nil, nil, types.ErrInvalidPool
	}

	client := rpc.New(url)
	tokenIsBase := !baseMint.Equals(solana.SolMint)
	var mint solana.PublicKey
	if tokenIsBase {
		mint = baseMint
	} else {
		mint = quoteMint
	}

	metaAddress, _, _ := solana.FindTokenMetadataAddress(mint)
	publicKeys := []solana.PublicKey{
		baseVault,
		quoteVault,
		mint,
		lpMint,
		metaAddress,
	}

	if withBalance {
		owner_ := solana.MPK(owner)
		ata, _, _ := solana.FindAssociatedTokenAddress(owner_, mint)
		publicKeys = append(publicKeys, owner_, ata)
	}

	accounts, err := client.GetMultipleAccountsWithOpts(
		ctx,
		publicKeys,
		&rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentProcessed},
	)
	if err != nil {
		return nil, nil, err
	}

	balance := &common.Balance{NativeBalance: big.NewInt(0), TokenBalance: big.NewInt(0)}
	if withBalance {
		if accounts.Value[5] != nil {
			balance.NativeBalance = new(big.Int).SetUint64(accounts.Value[5].Lamports)
		}

		if accounts.Value[6] != nil {
			var tokenAccount token.Account
			err = tokenAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[6].Data.GetBinary()))
			if err != nil {
				return nil, nil, err
			}
			balance.TokenBalance = new(big.Int).SetUint64(tokenAccount.Amount)
		}
	}

	var baseTokenAccount, quoteTokenAccount token.Account
	err = baseTokenAccount.UnmarshalWithDecoder(bin.NewBorshDecoder(accounts.Value[0].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}
	err = quoteTokenAccount.UnmarshalWithDecoder(bin.NewBorshDecoder(accounts.Value[1].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}

	var tokenMintAccount, lpMintAccount token.Mint
	err = tokenMintAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[2].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}
	err = lpMintAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[3].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}

	meta, err := common.MetadataDeserialize(accounts.Value[4].Data.GetBinary())
	if err != nil {
		return nil, balance, err
	}

	if quoteTokenAccount.Amount == 0 || baseTokenAccount.Amount == 0 {
		return nil, balance, types.ErrPoolCompleted
	}

	baseTokenAmount := decimal.NewFromBigInt(new(big.Int).SetUint64(baseTokenAccount.Amount), 0-int32(p.BaseDecimal))
	quoteTokenAmount := decimal.NewFromBigInt(new(big.Int).SetUint64(quoteTokenAccount.Amount), 0-int32(p.QuoteDecimal))

	var priceInSol decimal.Decimal
	var tokenReserve, solReserve *big.Int
	if tokenIsBase {
		tokenReserve = new(big.Int).SetUint64(baseTokenAccount.Amount)
		solReserve = new(big.Int).SetUint64(quoteTokenAccount.Amount)
		priceInSol = quoteTokenAmount.Div(baseTokenAmount)
	} else {
		tokenReserve = new(big.Int).SetUint64(quoteTokenAccount.Amount)
		solReserve = new(big.Int).SetUint64(baseTokenAccount.Amount)
		priceInSol = baseTokenAmount.Div(quoteTokenAmount)
	}

	totalSupply := decimal.NewFromUint64(tokenMintAccount.Supply).Div(decimal.New(1, int32(tokenMintAccount.Decimals)))

	return &common.GetSolPoolResponse{
		PriceInSol:            priceInSol,
		TotalSupply:           totalSupply,
		FreezeDisabled:        tokenMintAccount.FreezeAuthority == nil,
		Burnt:                 lpMintAccount.Supply == 0,
		MintAuthorityDisabled: tokenMintAccount.MintAuthority == nil,
		TokenReserve:          tokenReserve,
		SolReserve:            solReserve,
		Name:                  utils.TrimSpace(meta.Data.Name),
		Symbol:                utils.TrimSpace(meta.Data.Symbol),
		Decimals:              tokenMintAccount.Decimals,
		QuoteDecimals:         9,
		TokenAddress:          mint.String(),
		QuoteAddress:          solana.SolMint.String(),
		PoolAddress:           p.AmmPublicKey,
		MarketId:              p.MarketPublicKey,
		MarketProgramId:       p.MarketProgramID,
	}, balance, nil
}

func GeRaydiumPoolCLMM(ctx context.Context, url string, p *types.Pool, owner string, withBalance bool) (*common.GetSolPoolResponse, *common.Balance, error) {
	baseMint, err := solana.PublicKeyFromBase58(p.BaseMint)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	quoteMint, err := solana.PublicKeyFromBase58(p.QuoteMint)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	baseVault, err := solana.PublicKeyFromBase58(p.BaseVault)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	quoteVault, err := solana.PublicKeyFromBase58(p.QuoteVault)
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}
	if !baseMint.Equals(solana.SolMint) && !quoteMint.Equals(solana.SolMint) {
		return nil, nil, types.ErrInvalidPool
	}

	client := rpc.New(url)
	tokenIsBase := !baseMint.Equals(solana.SolMint)
	var mint solana.PublicKey
	if tokenIsBase {
		mint = baseMint
	} else {
		mint = quoteMint
	}

	metaAddress, _, _ := solana.FindTokenMetadataAddress(mint)
	publicKeys := []solana.PublicKey{
		baseVault,
		quoteVault,
		mint,
		metaAddress,
	}

	if withBalance {
		owner_ := solana.MPK(owner)
		ata, _, _ := solana.FindAssociatedTokenAddress(owner_, mint)
		publicKeys = append(publicKeys, owner_, ata)
	}

	accounts, err := client.GetMultipleAccountsWithOpts(
		ctx,
		publicKeys,
		&rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentProcessed},
	)
	if err != nil {
		return nil, nil, err
	}

	balance := &common.Balance{NativeBalance: big.NewInt(0), TokenBalance: big.NewInt(0)}
	if withBalance {
		if accounts.Value[4] != nil {
			balance.NativeBalance = new(big.Int).SetUint64(accounts.Value[4].Lamports)
		}

		if accounts.Value[5] != nil {
			var tokenAccount token.Account
			err = tokenAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[5].Data.GetBinary()))
			if err != nil {
				return nil, nil, err
			}
			balance.TokenBalance = new(big.Int).SetUint64(tokenAccount.Amount)
		}
	}

	var baseTokenAccount, quoteTokenAccount token.Account
	err = baseTokenAccount.UnmarshalWithDecoder(bin.NewBorshDecoder(accounts.Value[0].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}
	err = quoteTokenAccount.UnmarshalWithDecoder(bin.NewBorshDecoder(accounts.Value[1].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}

	var tokenMintAccount token.Mint
	err = tokenMintAccount.UnmarshalWithDecoder(bin.NewBinDecoder(accounts.Value[2].Data.GetBinary()))
	if err != nil {
		return nil, balance, err
	}

	var name, symbol string
	if v := accounts.Value[3]; v != nil && v.Data != nil {
		meta, err := common.MetadataDeserialize(v.Data.GetBinary())
		if err != nil {
			return nil, balance, err
		}
		name, symbol = meta.Data.Name, meta.Data.Symbol
	} else {
		mintInfo, err := GetRaydiumTokenMintInfo(ctx, mint.String())
		if err != nil {
			return nil, balance, err
		}
		name, symbol = mintInfo.Name, mintInfo.Symbol
	}

	if quoteTokenAccount.Amount == 0 || baseTokenAccount.Amount == 0 {
		return nil, balance, types.ErrPoolCompleted
	}

	baseTokenAmount := decimal.NewFromBigInt(new(big.Int).SetUint64(baseTokenAccount.Amount), 0-int32(p.BaseDecimal))
	quoteTokenAmount := decimal.NewFromBigInt(new(big.Int).SetUint64(quoteTokenAccount.Amount), 0-int32(p.QuoteDecimal))

	var priceInSol decimal.Decimal
	var tokenReserve, solReserve *big.Int
	if tokenIsBase {
		tokenReserve = new(big.Int).SetUint64(baseTokenAccount.Amount)
		solReserve = new(big.Int).SetUint64(quoteTokenAccount.Amount)
		priceInSol = quoteTokenAmount.Div(baseTokenAmount)
	} else {
		tokenReserve = new(big.Int).SetUint64(quoteTokenAccount.Amount)
		solReserve = new(big.Int).SetUint64(baseTokenAccount.Amount)
		priceInSol = baseTokenAmount.Div(quoteTokenAmount)
	}

	totalSupply := decimal.NewFromUint64(tokenMintAccount.Supply).Div(decimal.New(1, int32(tokenMintAccount.Decimals)))

	return &common.GetSolPoolResponse{
		PriceInSol:            priceInSol,
		TotalSupply:           totalSupply,
		FreezeDisabled:        tokenMintAccount.FreezeAuthority == nil,
		MintAuthorityDisabled: tokenMintAccount.MintAuthority == nil,
		TokenReserve:          tokenReserve,
		SolReserve:            solReserve,
		Name:                  utils.TrimSpace(name),
		Symbol:                utils.TrimSpace(symbol),
		Decimals:              tokenMintAccount.Decimals,
		QuoteDecimals:         9,
		TokenAddress:          mint.String(),
		QuoteAddress:          solana.SolMint.String(),
		PoolAddress:           p.AmmPublicKey,
		MarketId:              p.MarketPublicKey,
		MarketProgramId:       p.MarketProgramID,
	}, balance, nil
}

type MintInfo struct {
	ChainId    int           `json:"chainId"`
	Address    string        `json:"address"`
	ProgramId  string        `json:"programId"`
	LogoURI    string        `json:"logoURI"`
	Symbol     string        `json:"symbol"`
	Name       string        `json:"name"`
	Decimals   int           `json:"decimals"`
	Tags       []interface{} `json:"tags"`
	Extensions interface{}   `json:"extensions"`
}

func GetRaydiumTokenMintInfo(ctx context.Context, ca string) (*MintInfo, error) {
	url := fmt.Sprintf("%s/mint/ids?mints=%s", RaydiumApiV3, ca)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Id      string      `json:"id"`
		Success bool        `json:"success"`
		Data    []*MintInfo `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, types.ErrNotFound
	}
	return response.Data[0], nil
}

func GeRaydiumPool(ctx context.Context, url string, req *common.GetSolPoolRequest) (*common.GetSolPoolResponse, *common.Balance, error) {
	client := rpc.New(url)
	info, err := client.GetAccountInfo(ctx, solana.MPK(req.Pool))
	if err != nil {
		return nil, nil, err
	}

	var p Pool
	err = borsh.Deserialize(&p, info.Value.Data.GetBinary())
	if err != nil {
		return nil, nil, types.ErrInvalidPool
	}

	return GeRaydiumPoolP2(ctx, url, &types.Pool{
		AmmPublicKey:    req.Pool,
		MarketPublicKey: p.MarketId.String(),
		BaseMint:        p.BaseMint.String(),
		QuoteMint:       p.QuoteMint.String(),
		QuoteVault:      p.QuoteVault.String(),
		BaseVault:       p.BaseVault.String(),
		LpMint:          p.LpMint.String(),
		BaseDecimal:     uint8(p.BaseDecimal),
		QuoteDecimal:    uint8(p.QuoteDecimal),
		MarketProgramID: p.MarketProgramId.String(),
	}, req.Owner, req.WithBalance)
}

func GetRaydiumPoolByToken(ctx context.Context, url string, token solana.PublicKey, isBaseToken bool) (*types.Pool, error) {
	client := rpc.New(url)
	baseToken := lo.If(isBaseToken, token).Else(solana.SolMint)
	quoteToken := lo.If(isBaseToken, solana.SolMint).Else(token)

	result, err := client.GetProgramAccountsWithOpts(context.Background(), ProgramID, &rpc.GetProgramAccountsOpts{
		Commitment: rpc.CommitmentConfirmed,
		Encoding:   solana.EncodingBase64,
		Filters: []rpc.RPCFilter{
			{Memcmp: &rpc.RPCFilterMemcmp{Offset: 400, Bytes: baseToken.Bytes()}},
			{Memcmp: &rpc.RPCFilterMemcmp{Offset: 432, Bytes: quoteToken.Bytes()}},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	var pool Pool
	err = borsh.Deserialize(&pool, result[0].Account.Data.GetBinary())
	if err != nil {
		return nil, types.ErrInvalidPool
	}

	return &types.Pool{
		AmmPublicKey:    result[0].Pubkey.String(),
		MarketPublicKey: pool.MarketId.String(),
		MarketProgramID: pool.MarketProgramId.String(),
		BaseMint:        pool.BaseMint.String(),
		BaseVault:       pool.BaseVault.String(),
		BaseDecimal:     uint8(pool.BaseDecimal),
		QuoteMint:       pool.QuoteMint.String(),
		QuoteVault:      pool.QuoteVault.String(),
		QuoteDecimal:    uint8(pool.QuoteDecimal),
		LpMint:          pool.LpMint.String(),
		OpenTime:        int64(pool.PoolOpenTime),
		Dex:             0, // DexTypeRaydium
		Marked:          true,
	}, nil
}

func GetRaydiumCLMMPoolByToken(ctx context.Context, url string, token solana.PublicKey, isBaseToken bool) (*types.Pool, error) {
	client := rpc.New(url)
	baseToken := lo.If(isBaseToken, token).Else(solana.SolMint)
	quoteToken := lo.If(isBaseToken, solana.SolMint).Else(token)

	result, err := client.GetProgramAccountsWithOpts(context.Background(), CLMMProgramID, &rpc.GetProgramAccountsOpts{
		Commitment: rpc.CommitmentConfirmed,
		Encoding:   solana.EncodingBase64,
		Filters: []rpc.RPCFilter{
			{Memcmp: &rpc.RPCFilterMemcmp{Offset: 73, Bytes: quoteToken.Bytes()}},
			{Memcmp: &rpc.RPCFilterMemcmp{Offset: 105, Bytes: baseToken.Bytes()}},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	var pool ClmmPoolData
	var ammPK solana.PublicKey
	for _, account := range result {
		var tmp ClmmPoolData
		_ = borsh.Deserialize(&tmp, account.Account.Data.GetBinary())

		if tmp.Liquidity.Cmp(&pool.Liquidity) > 0 {
			pool = tmp
			ammPK = account.Pubkey
		}
	}

	return &types.Pool{
		AmmPublicKey:    ammPK.String(),
		MarketPublicKey: "",
		MarketProgramID: "",
		BaseMint:        pool.TokenMint1.String(),
		BaseVault:       pool.TokenVault1.String(),
		BaseDecimal:     pool.MintDecimals1,
		QuoteMint:       pool.TokenMint0.String(),
		QuoteVault:      pool.TokenVault0.String(),
		QuoteDecimal:    pool.MintDecimals0,
		LpMint:          "",
		OpenTime:        int64(pool.OpenTime),
		Status:          int(pool.Status),
		Dex:             0,
		Marked:          true,
	}, nil
}

type CreateSwapParam struct {
	MarketId solana.PublicKey

	Maker solana.PublicKey

	AmountIn     uint64
	MinAmountOut uint64

	Market      Market
	VaultSigner solana.PublicKey

	SourceAccount solana.PublicKey
	DestAccount   solana.PublicKey
}

func CreateSwapInstruction(param CreateSwapParam) solana.Instruction {
	data, err := borsh.Serialize(struct {
		Instruction  Instruction
		AmountIn     uint64
		MinAmountOut uint64
	}{
		Instruction:  InstructionSwap,
		AmountIn:     param.AmountIn,
		MinAmountOut: param.MinAmountOut,
	})

	if err != nil {
		panic(err)
	}

	ammId, _ := FindAmmId(param.MarketId)
	openOrdersAccount, _ := FindAmmOpenOrdersAccount(param.MarketId)
	targetOrdersAccount, _ := FindAmmTargetOrdersAccount(param.MarketId)
	poolCoinTokenAccount, _ := FindPoolCoinTokenAccount(param.MarketId)
	poolPcTokenAccount, _ := FindPoolPcTokenAccount(param.MarketId)

	return &solana.GenericInstruction{
		ProgID: ProgramID,
		AccountValues: solana.AccountMetaSlice{
			{PublicKey: solana.TokenProgramID, IsSigner: false, IsWritable: false},
			{PublicKey: ammId, IsSigner: false, IsWritable: true},
			{PublicKey: AuthorityV4, IsSigner: false, IsWritable: false},
			{PublicKey: openOrdersAccount, IsSigner: false, IsWritable: true},
			{PublicKey: targetOrdersAccount, IsSigner: false, IsWritable: true},
			{PublicKey: poolCoinTokenAccount, IsSigner: false, IsWritable: true},
			{PublicKey: poolPcTokenAccount, IsSigner: false, IsWritable: true},
			{PublicKey: OpenBookProgramID, IsSigner: false, IsWritable: false},
			{PublicKey: param.MarketId, IsSigner: false, IsWritable: true},
			{PublicKey: param.Market.Bids, IsSigner: false, IsWritable: true},
			{PublicKey: param.Market.Asks, IsSigner: false, IsWritable: true},
			{PublicKey: param.Market.EventQueue, IsSigner: false, IsWritable: true},
			{PublicKey: param.Market.BaseVault, IsSigner: false, IsWritable: true},
			{PublicKey: param.Market.QuoteVault, IsSigner: false, IsWritable: true},
			{PublicKey: param.VaultSigner, IsSigner: false, IsWritable: false},
			{PublicKey: param.SourceAccount, IsSigner: false, IsWritable: true},
			{PublicKey: param.DestAccount, IsSigner: false, IsWritable: true},
			{PublicKey: param.Maker, IsSigner: true, IsWritable: true},
		},
		DataBytes: data,
	}
}

func CreateIdempotentInstruction(mint, wallet solana.PublicKey) solana.Instruction {
	ata, _, _ := solana.FindAssociatedTokenAddress(wallet, mint)

	return &solana.GenericInstruction{
		ProgID: solana.SPLAssociatedTokenAccountProgramID,
		AccountValues: solana.AccountMetaSlice{
			{PublicKey: wallet, IsSigner: true, IsWritable: true},
			{PublicKey: ata, IsSigner: false, IsWritable: true},
			{PublicKey: wallet, IsSigner: false, IsWritable: false},
			{PublicKey: mint, IsSigner: false, IsWritable: false},
			{PublicKey: solana.SystemProgramID, IsSigner: false, IsWritable: false},
			{PublicKey: solana.TokenProgramID, IsSigner: false, IsWritable: false},
		},
	}
}
