package sol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/meme-bots/go-web3/types"
	"github.com/shopspring/decimal"
)

type (
	Token struct {
		Address string `json:"address"`
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
	}

	PriceChanged struct {
		M5  decimal.Decimal `json:"m5"`
		H1  decimal.Decimal `json:"h1"`
		H6  decimal.Decimal `json:"h6"`
		H24 decimal.Decimal `json:"h24"`
	}

	DexScreenerPair struct {
		// PoolAddress string `json:"pairAddress"`
		BaseToken Token `json:"baseToken"`
		// QuoteToken  Token        `json:"quoteToken"`
		PriceChange PriceChanged `json:"priceChange"`
		// DexID       string       `json:"dexId"`
		// PriceNative string       `json:"priceNative"`
		// PriceUSD    string       `json:"priceUsd"`
	}

	QueryDexScreenerResponse struct {
		Pairs []DexScreenerPair `json:"pairs"`
	}

	PumpFunFrame struct {
		Close     decimal.Decimal `json:"close"`
		Timestamp uint64          `json:"timestamp"`
	}
)

const MaxDuration = 5 * time.Minute

func QueryDexScreener(address string) (*DexScreenerPair, error) {
	url := "https://api.dexscreener.com/latest/dex/search/?q=" + address
	ret, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return nil, err
	}

	var resp QueryDexScreenerResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Pairs) > 0 {
		return &resp.Pairs[0], nil
	}

	return nil, types.ErrNotFound
}

func QueryDexScreenerWithCache(cached *cache.Cache[[]byte], address string) (*DexScreenerPair, error) {
	ctx := context.Background()
	key := "QueryDexScreener:" + address

	data, ttl, err := cached.GetWithTTL(ctx, key)
	if err == nil {
		if ttl < MaxDuration {
			var pair DexScreenerPair
			err = json.Unmarshal(data, &pair)
			if err != nil {
				return nil, err
			}
			return &pair, nil
		}
	}

	pair, err := QueryDexScreener(address)
	if err != nil {
		return nil, err
	}

	data, err = json.Marshal(pair)
	if err == nil {
		cached.Set(ctx, key, data, store.WithExpiration(MaxDuration))
	}

	return pair, nil
}

func QueryPumpFun(address string) (*types.PriceHistorical, error) {
	url := fmt.Sprintf("https://frontend-api.pump.fun/candlesticks/%s?offset=0&limit=288&is_5_min=true", address)
	ret, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return nil, err
	}

	frames := make([]PumpFunFrame, 0)
	err = json.Unmarshal(body, &frames)
	if err != nil {
		return nil, err
	}

	historical := &types.PriceHistorical{
		M5:  decimal.Zero,
		H1:  decimal.Zero,
		H6:  decimal.Zero,
		H24: decimal.Zero,
	}

	cnt := len(frames)
	if cnt >= 1 {
		historical.M5 = frames[cnt-1].Close
	}
	if cnt >= 12 {
		historical.H1 = frames[cnt-12].Close
	}
	if cnt >= 72 {
		historical.H6 = frames[cnt-72].Close
	}
	if cnt >= 288 {
		historical.H24 = frames[cnt-288].Close
	}

	return historical, nil
}

func QueryPumpFunWithCache(cached *cache.Cache[[]byte], address string) (*types.PriceHistorical, error) {
	ctx := context.Background()
	key := "QueryPumpFun:" + address

	data, ttl, err := cached.GetWithTTL(ctx, key)
	if err == nil {
		if ttl < MaxDuration {
			var historical types.PriceHistorical
			err = json.Unmarshal(data, &historical)
			if err != nil {
				return nil, err
			}
			return &historical, nil
		}
	}

	historical, err := QueryPumpFun(address)
	if err != nil {
		return nil, err
	}

	data, err = json.Marshal(historical)
	if err == nil {
		cached.Set(ctx, key, data, store.WithExpiration(MaxDuration))
	}

	return historical, nil
}
