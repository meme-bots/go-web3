package sol

import (
	"context"
	"errors"
	"sync"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/utils"
	"github.com/shopspring/decimal"
)

type (
	watcherState uint8

	Watcher struct {
		client        *rpc.Client
		price         decimal.Decimal
		priceLock     sync.RWMutex
		hash          solana.Hash
		hashUpdatedAt time.Time
		hashLock      sync.RWMutex
		withBlockHash bool

		ctx          context.Context
		cancel       context.CancelFunc
		subprocesses utils.Subprocesses

		stateMu sync.Mutex
		state   watcherState
	}
)

const (
	_ watcherState = iota
	watcherStatePending
	watcherStateOpen
	watcherStateClosed
)

var (
	solVaultAddress  = solana.MPK("876Z9waBygfzUrwwKFfnRcc7cfY4EQf6Kz1w7GRgbVYW")
	usdtVaultAddress = solana.MPK("CB86HtaqpXbNWbq67L18y5x2RhqoJ6smb7xHUcyWdQAQ")
)

func NewWatcher(url string, withBlockHash bool) (*Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		client:        rpc.New(url),
		price:         decimal.Zero,
		hash:          solana.Hash{},
		withBlockHash: withBlockHash,
		ctx:           ctx,
		cancel:        cancel,
		subprocesses:  utils.Subprocesses{},
		stateMu:       sync.Mutex{},
		state:         watcherStatePending,
	}, nil
}

func (w *Watcher) Start() error {
	succeed := false
	defer func() {
		if !succeed {
			w.Close()
		}
	}()

	w.stateMu.Lock()
	defer w.stateMu.Unlock()

	if w.state != watcherStatePending {
		return errors.New("cannot Start() watcher that has already been started")
	}

	w.state = watcherStateOpen

	w.subprocesses.Go(func() {
		w.WatchSolPrice(time.Minute)
	})

	if w.withBlockHash {
		w.subprocesses.Go(func() {
			w.WatchBlockHash(time.Second)
		})
	}

	succeed = true
	return nil
}

func (w *Watcher) Close() error {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()

	if w.state != watcherStateOpen {
		return errors.New("cannot Close() host that isn't open")
	}

	w.state = watcherStateClosed
	w.cancel()
	w.subprocesses.Wait()
	return nil
}

func (w *Watcher) QuerySolPrice() (decimal.Decimal, error) {
	accounts, err := w.client.GetMultipleAccountsWithOpts(
		w.ctx,
		[]solana.PublicKey{solVaultAddress, usdtVaultAddress},
		&rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentConfirmed},
	)
	if err != nil {
		return decimal.Zero, err
	}

	var wsolVault, usdtVault token.Account
	err = wsolVault.UnmarshalWithDecoder(bin.NewBorshDecoder(accounts.Value[0].Data.GetBinary()))
	if err != nil {
		return decimal.Zero, err
	}

	err = usdtVault.UnmarshalWithDecoder(bin.NewBorshDecoder(accounts.Value[1].Data.GetBinary()))
	if err != nil {
		return decimal.Zero, err
	}

	wsolAmount := decimal.NewFromUint64(wsolVault.Amount).Div(decimal.NewFromInt(1e9))
	usdtAmount := decimal.NewFromUint64(usdtVault.Amount).Div(decimal.NewFromInt(1e6))
	return usdtAmount.Div(wsolAmount), nil
}

func (w *Watcher) QueryBlockHash() (solana.Hash, error) {
	recentBlock, err := w.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return solana.Hash{}, err
	}
	return recentBlock.Value.Blockhash, nil
}

func (w *Watcher) WatchSolPrice(interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
		case <-w.ctx.Done():
			return
		}

		price, err := w.QuerySolPrice()
		if err == nil {
			w.priceLock.Lock()
			w.price = price
			w.priceLock.Unlock()
		}
	}
}

func (w *Watcher) WatchBlockHash(interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
		case <-w.ctx.Done():
			return
		}

		hash, err := w.QueryBlockHash()
		if err == nil {
			w.hashLock.Lock()
			w.hash = hash
			w.hashUpdatedAt = time.Now()
			w.hashLock.Unlock()
		}
	}
}

func (w *Watcher) GetSolPrice() decimal.Decimal {
	var price decimal.Decimal
	w.priceLock.RLock()
	price = w.price
	w.priceLock.RUnlock()
	return price
}

func (w *Watcher) GetRecentBlockHash() (solana.Hash, bool) {
	if !w.withBlockHash {
		return solana.Hash{}, false
	}

	w.hashLock.RLock()
	defer w.hashLock.RUnlock()
	since := time.Since(w.hashUpdatedAt).Milliseconds()
	if since > 3000 {
		return solana.Hash{}, false
	}
	return solana.HashFromBytes(w.hash[:]), true
}
