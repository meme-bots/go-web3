package sol

import (
	"context"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/meme-bots/go-web3/sol/pumpfun"
	"github.com/meme-bots/go-web3/sol/raydium"
	"github.com/near/borsh-go"
	"testing"
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

func TestSolana_GetRaydiumCLMMPoolByToken(t *testing.T) {
	r := "https://api.mainnet-beta.solana.com"
	token := solana.MPK("HeLp6NuQkmYB4pYWo2zYs22mESHXPQYzXbB8n4V98jwC")
	pool, err := raydium.GetRaydiumCLMMPoolByToken(context.Background(), r, token, true)
	t.Log(err)
	t.Log(pool)

	ret, balance, err := raydium.GeRaydiumPoolCLMM(context.Background(), r, pool, "Gmgy14zk3eNwyNWKUS5qJQcmRDGFirwFyGkAXxTc3LFZ", false)
	t.Log(ret)
	t.Log(balance)
}
