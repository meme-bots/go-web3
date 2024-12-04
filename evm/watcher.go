package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/meme-bots/go-web3/evm/chainlink"
	"github.com/meme-bots/go-web3/utils"
	"github.com/shopspring/decimal"
)

type (
	watcherState uint8

	Watcher struct {
		client       *ethclient.Client
		ethPrice     decimal.Decimal
		ethPriceLock sync.RWMutex
		gasPrice     *big.Int
		gasPriceLock sync.RWMutex
		oracle       *chainlink.AggregatorV3Interface

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

func NewWatcher(url string, ethPriceOracle common.Address) (*Watcher, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	oracle, err := chainlink.NewAggregatorV3Interface(ethPriceOracle, client)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		client:   client,
		ethPrice: decimal.Zero,
		gasPrice: big.NewInt(0),
		oracle:   oracle,
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
		w.WatchETHPrice(time.Second * 30)
	})

	w.subprocesses.Go(func() {
		w.WatchGasPrice(time.Second * 10)
	})

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

func (w *Watcher) WatchETHPrice(interval time.Duration) {
	decimals, err := w.oracle.Decimals(&bind.CallOpts{})
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		select {
		case <-time.After(interval):
		case <-w.ctx.Done():
			return
		}

		data, err := w.oracle.LatestRoundData(&bind.CallOpts{})
		if err != nil {
			fmt.Println(err)
		} else {
			w.ethPriceLock.Lock()
			w.ethPrice = decimal.NewFromBigInt(data.Answer, 0-int32(decimals))
			w.ethPriceLock.Unlock()
		}
	}
}

func (w *Watcher) WatchGasPrice(interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
		case <-w.ctx.Done():
			return
		}

		price, err := w.client.SuggestGasPrice(context.Background())
		if err != nil {
			fmt.Println(err)
		} else {
			w.gasPriceLock.Lock()
			w.gasPrice = price
			w.gasPriceLock.Unlock()
		}
	}
}

func (w *Watcher) GetETHPrice() decimal.Decimal {
	var price decimal.Decimal
	w.ethPriceLock.RLock()
	price = w.ethPrice.Copy()
	w.ethPriceLock.RUnlock()
	return price
}

func (w *Watcher) GetGasPrice() *big.Int {
	var price *big.Int
	w.gasPriceLock.RLock()
	price = new(big.Int).Set(w.gasPrice)
	w.gasPriceLock.RUnlock()
	return price
}
