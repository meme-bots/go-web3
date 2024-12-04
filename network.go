package goweb3

import (
	"context"

	"github.com/meme-bots/go-web3/evm"
	"github.com/meme-bots/go-web3/sol"
	"github.com/meme-bots/go-web3/types"
)

func NewNetwork(ctx context.Context, cfg types.Config) (types.NetworkInterface, error) {
	if cfg.Type == types.NetworkTypeSol {
		return sol.NewSolana(ctx, &cfg)
	} else if cfg.Type == types.NetworkTypeEVM {
		return evm.NewEVM(ctx, &cfg)
	}
	return nil, types.ErrNotImplemented
}
