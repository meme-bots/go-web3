package sol

import (
	"context"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/sol/pumpfun"
	"github.com/near/borsh-go"
)

func TestSolana_GetGlobal(t *testing.T) {
	c := rpc.New("") // TODO
	account, err := c.GetAccountInfoWithOpts(
		context.Background(),
		pumpfun.GlobalPubKey,
		&rpc.GetAccountInfoOpts{Commitment: rpc.CommitmentConfirmed},
	)
	if err != nil {
		t.Fatal(err)
	}
	var global pumpfun.Global
	err = borsh.Deserialize(&global, account.Value.Data.GetBinary())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("global: %+v", global)
	t.Logf("token: %d", pumpfun.GetInitialBuyPrice(&global, 30000000000))
}
