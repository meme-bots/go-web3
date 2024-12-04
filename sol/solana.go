package sol

import (
	"context"
	"errors"
	"math/big"
	"regexp"
	"slices"
	"strings"

	"github.com/eko/gocache/lib/v4/cache"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/meme-bots/go-web3/sol/common"
	"github.com/meme-bots/go-web3/sol/pumpfun"
	"github.com/meme-bots/go-web3/sol/raydium"
	"github.com/meme-bots/go-web3/types"
	"github.com/meme-bots/go-web3/utils"
	"github.com/near/borsh-go"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type Solana struct {
	ctx     context.Context
	cfg     *types.Config
	watcher *Watcher
	client  *ws.Client
	cache   *cache.Cache[[]byte]
}

func NewSolana(
	ctx context.Context,
	cfg *types.Config,
) (*Solana, error) {
	watcher, err := NewWatcher(cfg.RPC, cfg.WatchBlockHash)
	if err != nil {
		return nil, err
	}

	client, err := ws.Connect(ctx, cfg.WSRPC)
	if err != nil {
		return nil, err
	}

	cache, err := utils.NewCache()
	if err != nil {
		return nil, err
	}

	return &Solana{
		ctx:     ctx,
		cfg:     cfg,
		watcher: watcher,
		client:  client,
		cache:   cache,
	}, nil
}

func (s *Solana) Start() error {
	return s.watcher.Start()
}

func (s *Solana) Close() error {
	return s.watcher.Close()
}

func (s *Solana) GetType() int {
	return types.NetworkTypeSol
}

func (s *Solana) GetTypeSymbol() string {
	return "SOL"
}

func (s *Solana) GetNativeTokenSymbol() string {
	return s.cfg.NativeTokenSymbol
}

func (s *Solana) GetNativeTokenDecimals() uint8 {
	return s.cfg.NativeTokenDecimals
}

func (s *Solana) GetNativeTokenPrice() decimal.Decimal {
	return s.watcher.GetSolPrice()
}

func (s *Solana) GetMaxMultiSendCount() int {
	return MaxRecipientCount
}

func (s *Solana) GetBalance(req *types.GetBalanceRequest) (*big.Int, error) {
	c := rpc.New(s.cfg.RPC)
	balance, err := c.GetBalance(s.ctx, solana.MPK(req.Address), rpc.CommitmentConfirmed)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(balance.Value), nil
}

func (s *Solana) GetTokenBalance(req *types.GetTokenBalanceRequest) (*big.Int, error) {
	c := rpc.New(s.cfg.RPC)
	mint := solana.MPK(req.Token)
	ret, err := c.GetTokenAccountsByOwner(s.ctx, solana.MPK(req.Owner), &rpc.GetTokenAccountsConfig{Mint: &mint}, &rpc.GetTokenAccountsOpts{Commitment: rpc.CommitmentConfirmed})
	if err != nil {
		return nil, err
	}
	if len(ret.Value) == 0 {
		return big.NewInt(0), nil
	}

	var account token.Account
	err = account.UnmarshalWithDecoder(bin.NewBorshDecoder(ret.Value[0].Account.Data.GetBinary()))
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetUint64(account.Amount), nil
}

func (s *Solana) QueryPool(req *types.QueryPoolRequest) (*types.Pool, error) {
	var p1, p2, p3 *types.Pool
	sub := utils.Subprocesses{}
	mint := solana.MPK(req.Token)

	sub.Go(func() {
		p1, _ = raydium.GetRaydiumPoolByToken(context.Background(), s.cfg.RPC, mint, true)
	})
	sub.Go(func() {
		p2, _ = raydium.GetRaydiumPoolByToken(context.Background(), s.cfg.RPC, mint, false)
	})
	sub.Go(func() {
		p3, _ = pumpfun.GetPumpFunPoolByToken(context.Background(), s.cfg.RPC, mint)
	})

	sub.Wait()

	p := lo.If(p1 != nil, p1).ElseIf(p2 != nil, p2).ElseIf(p3 != nil, p3).Else(nil)
	if p == nil {
		return nil, types.ErrInvalidPool
	}

	return p, nil
}

func (s *Solana) GetPool(req *types.GetPoolRequest) (*types.GetPoolResponse, error) {
	var ret *common.GetSolPoolResponse
	var err error
	var dexID int = 0

	token := req.Token
	valid := s.CheckAddress(token)
	if !valid {
		if strings.Contains(req.URL, "dexscreener.com") {
			pair, err := QueryDexScreener(token)
			if err != nil {
				return nil, err
			}
			token = pair.BaseToken.Address
			dexID = 0
		} else {
			return nil, types.ErrInvalidPool
		}
	}

	var balance *common.Balance
	p, err := s.QueryPool(&types.QueryPoolRequest{Token: token})
	if err != nil {
		return nil, err
	}

	if p.Dex == 0 {
		dexID = 0
	} else if p.Dex == 1 && p.Status == 0 {
		dexID = 1
	} else {
		return nil, types.ErrInvalidPool
	}

	if dexID == 1 {
		ret, balance, err = pumpfun.GetPumpFunPool(s.ctx, s.cfg.RPC, &common.GetSolPoolRequest{Token: token, Owner: req.Owner, WithBalance: true})
	} else {
		ret, balance, err = raydium.GeRaydiumPoolP2(s.ctx, s.cfg.RPC, p, req.Owner, true)
	}
	if err != nil {
		return nil, err
	}

	nativeTokenPrice := s.GetNativeTokenPrice()
	priceInUSD := ret.PriceInSol.Mul(nativeTokenPrice)

	return &types.GetPoolResponse{
		NativeTokenPrice:      nativeTokenPrice,
		PriceInNativeToken:    ret.PriceInSol,
		PriceInUSD:            priceInUSD,
		TotalSupply:           ret.TotalSupply,
		MarketCap:             ret.TotalSupply.Mul(priceInUSD),
		FreezeDisabled:        ret.FreezeDisabled,
		Burnt:                 ret.Burnt,
		MintAuthorityDisabled: ret.MintAuthorityDisabled,
		TokenReserve:          ret.TokenReserve,
		QuoteReserve:          ret.SolReserve,
		Name:                  ret.Name,
		Symbol:                ret.Symbol,
		Decimals:              ret.Decimals,
		QuoteDecimals:         ret.QuoteDecimals,
		TokenAddress:          ret.TokenAddress,
		QuoteAddress:          ret.QuoteAddress,
		PoolAddress:           ret.PoolAddress,
		DexID:                 dexID,
		MarketId:              ret.MarketId,
		MarketProgramId:       ret.MarketProgramId,
		NativeBalance:         balance.NativeBalance,
		TokenBalance:          balance.TokenBalance,
	}, nil
}

func (s *Solana) WatchTransaction(req *types.WatchTransactionRequest) (interface{}, error) {
	sig, err := solana.SignatureFromBase58(req.TxHash)
	if err != nil {
		return nil, err
	}

	sub, err := s.client.SignatureSubscribe(sig, rpc.CommitmentProcessed)
	for err != nil {
		s.WsReconnect()
		sub, err = s.client.SignatureSubscribe(sig, rpc.CommitmentProcessed)
	}
	defer sub.Unsubscribe()

	result, err := sub.RecvWithTimeout(req.Duration)
	if err != nil {
		return nil, err
	}

	if result.Value.Err != nil {
		errMap := result.Value.Err.(map[string]interface{})
		if _, ok := errMap["InstructionError"]; ok {
			return nil, types.ErrInstructionFailed
		}
		return nil, types.ErrTransactionFailed
	}
	return nil, nil
}

func (s *Solana) GetTransaction(req *types.GetTransactionRequest) (*types.GetTransactionResponse, error) {
	c := rpc.New(s.cfg.RPC)
	tx, err := c.GetTransaction(s.ctx, solana.MustSignatureFromBase58(req.TxHash), &rpc.GetTransactionOpts{Commitment: rpc.CommitmentConfirmed})
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return nil, types.ErrTxNotLand
		}
		return nil, err
	}

	if tx.Meta.Err != nil {
		for _, logMessage := range tx.Meta.LogMessages {
			if strings.Contains(logMessage, "TooMuchSolRequired") || strings.Contains(logMessage, "exceeds desired slippage limit") {
				return nil, types.ErrSlippage
			}
		}
		return nil, types.ErrTransactionFailed
	}

	transaction, err := tx.Transaction.GetTransaction()
	if err != nil {
		return nil, err
	}

	var solSwapped, tokenSwapped *big.Int
	var botFee uint64
	for i, instruction := range transaction.Message.Instructions {
		programID, _ := transaction.Message.Account(instruction.ProgramIDIndex)

		if programID.Equals(raydium.ProgramID) && instruction.Data[0] == uint8(raydium.InstructionSwap) {
			ata, _, _ := solana.FindAssociatedTokenAddress(solana.MPK(req.Owner), solana.MPK(req.Token))
			ataIndex, _ := transaction.GetAccountIndex(ata)
			solSwapped, tokenSwapped = raydium.ParseSwapInstruction(tx, uint16(i), ataIndex)
		} else if programID.Equals(pumpfun.ProgramID) {
			if slices.Equal(instruction.Data[:8], pumpfun.Instruction_Buy[:]) {
				solSwapped, tokenSwapped = pumpfun.ParseBuyInstruction(tx, uint16(i))
			} else if slices.Equal(instruction.Data[:8], pumpfun.Instruction_Sell[:]) {
				solSwapped, tokenSwapped = pumpfun.ParseSellInstruction(tx, uint16(i))
			}
		}

		if programID.Equals(solana.SystemProgramID) && uint32(instruction.Data[0]) == system.Instruction_Transfer {
			feeRecipientIdx, err := transaction.GetAccountIndex(solana.MPK(req.FeeRecipient))
			if err == nil && feeRecipientIdx == instruction.Accounts[1] {
				var transfer system.Transfer
				_ = transfer.UnmarshalWithDecoder(bin.NewBorshDecoder(instruction.Data[4:]))
				botFee = *transfer.Lamports
			}
		}
	}

	return &types.GetTransactionResponse{
		BalanceChanged: solSwapped,
		TokenChanged:   tokenSwapped,
		Fee:            new(big.Int).SetUint64(tx.Meta.Fee),
		BotFee:         new(big.Int).SetUint64(botFee),
	}, nil
}

func (s *Solana) CheckAddress(text string) bool {
	reg, _ := regexp.Compile("^[1-9A-HJ-NP-Za-km-z]{32,44}$")
	if !reg.MatchString(text) {
		return false
	}
	_, err := solana.PublicKeyFromBase58(text)
	return err == nil
}

func (s *Solana) CheckNormalizedAddress(text string) bool {
	reg, _ := regexp.Compile("^[1-9a-z]{32,44}$")
	return reg.MatchString(text)
}

func (s *Solana) GetAddressFromInput(text string) string {
	regexs := []string{
		`^([1-9A-HJ-NP-Za-km-z]{32,44})$`,
		`^https://birdeye.so/token/([1-9A-HJ-NP-Za-km-z]{32,44})$`,
		`^https://birdeye.so/token/([1-9A-HJ-NP-Za-km-z]{32,44})\?.*$`,
		`^https://pump.fun/([1-9A-HJ-NP-Za-km-z]{32,44})$`,
		//`^https://dexscreener.com/solana/([1-9A-HJ-NP-Za-km-z]{32,44})$`,
		`^https://dexscreener.com/solana/([1-9a-z]{32,44})$`,
	}

	for _, reg := range regexs {
		re := regexp.MustCompile(reg)
		matches := re.FindStringSubmatch(text)
		if len(matches) == 2 {
			return matches[1]
		}
	}

	return ""
}

func (s *Solana) Withdraw(to string, amount decimal.Decimal, privateKey string) (string, error) {
	recipient, err := solana.PublicKeyFromBase58(to)
	if err != nil {
		return "", err
	}

	pk, err := solana.PrivateKeyFromBase58(privateKey)
	if err != nil {
		return "", err
	}

	recentBlockHash, _ := s.watcher.GetRecentBlockHash()

	signature, err := SendTransfer(
		s.ctx,
		s.cfg.RPC,
		recipient,
		amount.Mul(decimal.New(1, 9)).BigInt().Uint64(),
		pk,
		recentBlockHash,
	)
	if err != nil {
		return "", err
	}
	return signature.String(), nil
}

func (s *Solana) Transact(req *types.Transact, feeRecipient_ string, feeRatio uint64, privateKey string) (*types.TransactResponse, error) {
	c := rpc.New(s.cfg.RPC)
	feeRecipient := solana.MPK(feeRecipient_)
	pk := solana.MustPrivateKeyFromBase58(privateKey)

	var err error
	var signature solana.Signature
	var tokenMint solana.PublicKey
	buy := solana.MPK(req.TokenIn).Equals(solana.SolMint)
	if buy {
		tokenMint = solana.MPK(req.TokenOut)
	} else {
		tokenMint = solana.MPK(req.TokenIn)
	}

	var tokenBalance uint64 = 0
	var positionClosed bool = false
	var accounts *rpc.GetMultipleAccountsResult

	recentBlockHash, _ := s.watcher.GetRecentBlockHash()

	ata, _, _ := solana.FindAssociatedTokenAddress(pk.PublicKey(), tokenMint)
	if req.Dex == 0 { // raydium
		accounts, err = c.GetMultipleAccountsWithOpts(
			s.ctx,
			[]solana.PublicKey{ata, solana.MPK(req.MarketId)},
			&rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentConfirmed},
		)
		if err != nil {
			return nil, err
		}
		var market raydium.Market
		err = borsh.Deserialize(&market, accounts.Value[1].Data.GetBinary())
		if err != nil {
			return nil, err
		}
		createAta := accounts.Value[0] == nil
		if !createAta {
			var tokenAccount token.Account
			err = borsh.Deserialize(&tokenAccount, accounts.Value[0].Data.GetBinary())
			if err != nil {
				return nil, err
			}
			tokenBalance = tokenAccount.Amount
		}

		if buy {
			signature, err = raydium.SendBuy(
				s.ctx,
				s.cfg.RPC,
				solana.MPK(req.MarketId),
				solana.MPK(req.MarketProgramId),
				feeRecipient,
				req.InAmount.Uint64(),
				uint64(req.SlipPage),
				req.QuoteReserve.Uint64(),
				req.TokenReserve.Uint64(),
				req.Gas.Uint64(),
				feeRatio,
				req.Tip.Uint64(),
				market,
				createAta,
				pk,
				recentBlockHash,
			)
		} else {
			positionClosed = tokenBalance == req.InAmount.Uint64()
			signature, err = raydium.SendSell(
				s.ctx,
				s.cfg.RPC,
				solana.MPK(req.MarketId),
				solana.MPK(req.MarketProgramId),
				feeRecipient,
				req.InAmount.Uint64(),
				uint64(req.SlipPage),
				req.QuoteReserve.Uint64(),
				req.TokenReserve.Uint64(),
				req.Gas.Uint64(),
				feeRatio,
				req.Tip.Uint64(),
				market,
				positionClosed,
				pk,
				recentBlockHash,
			)
		}
	} else { // pumpfun
		bondingCurvePubKey := pumpfun.FindBondingCurve(tokenMint)
		accounts, err = c.GetMultipleAccountsWithOpts(
			s.ctx,
			[]solana.PublicKey{ata, bondingCurvePubKey},
			&rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentConfirmed},
		)
		if err != nil {
			return nil, err
		}
		var bondingCurve pumpfun.BondingCurve
		err = borsh.Deserialize(&bondingCurve, accounts.Value[1].Data.GetBinary())
		if err != nil {
			return nil, err
		}
		createAta := accounts.Value[0] == nil

		if !createAta {
			var tokenAccount token.Account
			err = borsh.Deserialize(&tokenAccount, accounts.Value[0].Data.GetBinary())
			if err != nil {
				return nil, err
			}
			tokenBalance = tokenAccount.Amount
		}

		if buy {
			signature, err = pumpfun.SendBuy(
				s.ctx,
				s.cfg.RPC,
				tokenMint,
				feeRecipient,
				req.InAmount.Uint64(),
				uint64(req.SlipPage),
				req.Gas.Uint64(),
				feeRatio,
				req.Tip.Uint64(),
				bondingCurve,
				createAta,
				pk,
				recentBlockHash,
			)
		} else {
			positionClosed = tokenBalance == req.InAmount.Uint64()
			signature, err = pumpfun.SendSell(
				s.ctx,
				s.cfg.RPC,
				tokenMint,
				feeRecipient,
				req.InAmount.Uint64(),
				uint64(req.SlipPage),
				req.Gas.Uint64(),
				feeRatio,
				req.Tip.Uint64(),
				bondingCurve,
				positionClosed,
				pk,
				recentBlockHash,
			)
		}
	}
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "AccountNotInitialized") {
			return nil, types.ErrAccountNotInitialized
		} else if strings.Contains(errStr, "TooMuchSolRequired") || strings.Contains(errStr, "exceeds desired slippage limit") {
			return nil, types.ErrSlippage
		}

		return nil, err
	}
	return &types.TransactResponse{
		TxHash:              signature.String(),
		InitialTokenBalance: new(big.Int).SetUint64(tokenBalance),
		PositionClosed:      positionClosed,
	}, nil
}

func (s *Solana) WsReconnect() {
	conn, err := ws.Connect(context.Background(), s.cfg.WSRPC)
	for err != nil {
		conn, err = ws.Connect(context.Background(), s.cfg.WSRPC)
	}
	s.client = conn
}

func (s *Solana) GetBaseGas() *big.Int {
	return new(big.Int).SetUint64(5000)
}

func (s *Solana) SendNative(bill *types.TransferBill, privateKey string) (string, error) {
	recentBlockHash, _ := s.watcher.GetRecentBlockHash()
	signature, err := SendTransfer(
		s.ctx,
		s.cfg.RPC,
		solana.MustPublicKeyFromBase58(bill.Recipient),
		bill.Amount.Uint64(),
		solana.MustPrivateKeyFromBase58(privateKey),
		recentBlockHash,
	)
	if err != nil {
		return "", err
	}
	return signature.String(), nil
}

func (s *Solana) SendNativeBatch(bills []*types.TransferBill, privateKey string) (string, error) {
	recentBlockHash, _ := s.watcher.GetRecentBlockHash()
	signature, err := SendTransferBatch(
		s.ctx,
		s.cfg.RPC,
		solana.MustPrivateKeyFromBase58(privateKey),
		bills,
		recentBlockHash,
	)
	if err != nil {
		return "", err
	}
	return signature.String(), nil
}
