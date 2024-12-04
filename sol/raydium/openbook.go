package raydium

import (
	"bytes"
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
	"github.com/near/borsh-go"
)

var (
	OpenBookProgramID = solana.MPK("srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX")
)

type Market struct {
	X [5]uint8
	Y [8]uint8

	OwnerAddress     solana.PublicKey
	VaultSignerNonce uint64

	BaseMint  solana.PublicKey
	QuoteMint solana.PublicKey

	BaseVault         solana.PublicKey
	BaseDepositsTotal uint64
	BaseFeesAccrued   uint64

	QuoteVault         solana.PublicKey
	QuoteDepositsTotal uint64
	QuoteFeesAccrued   uint64

	QuoteDustThreshold uint64

	RequestQueue solana.PublicKey
	EventQueue   solana.PublicKey

	Bids solana.PublicKey
	Asks solana.PublicKey

	BaseLotSize  uint64
	QuoteLotSize uint64

	FeeRateBps uint64

	ReferrerRebatesAccrued uint64

	Z [7]uint8
}

func MarketDeserialize(data []byte) (Market, error) {
	var market Market
	err := borsh.Deserialize(&market, data)
	return market, err
}

func FindAmmId(market solana.PublicKey) (solana.PublicKey, error) {
	ammId, _, err := solana.FindProgramAddress(
		[][]byte{
			ProgramID.Bytes(),
			market.Bytes(),
			[]byte("amm_associated_seed"),
		},
		ProgramID,
	)
	return ammId, err
}

func FindPoolCoinTokenAccount(market solana.PublicKey) (solana.PublicKey, error) {
	ammId, _, err := solana.FindProgramAddress(
		[][]byte{
			ProgramID.Bytes(),
			market.Bytes(),
			[]byte("coin_vault_associated_seed"),
		},
		ProgramID,
	)
	return ammId, err
}

func FindPoolPcTokenAccount(market solana.PublicKey) (solana.PublicKey, error) {
	ammId, _, err := solana.FindProgramAddress(
		[][]byte{
			ProgramID.Bytes(),
			market.Bytes(),
			[]byte("pc_vault_associated_seed"),
		},
		ProgramID,
	)
	return ammId, err
}

func FindLpMint(market solana.PublicKey) (solana.PublicKey, error) {
	ammId, _, err := solana.FindProgramAddress(
		[][]byte{
			ProgramID.Bytes(),
			market.Bytes(),
			[]byte("lp_mint_associated_seed"),
		},
		ProgramID,
	)
	return ammId, err
}

func FindVaultSigner(nonce uint64, marketID, marketProgramID solana.PublicKey) (solana.PublicKey, error) {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.LittleEndian, nonce)

	vaultSigner, err := solana.CreateProgramAddress(
		[][]byte{marketID.Bytes(), buffer.Bytes()},
		marketProgramID)
	return vaultSigner, err
}

func FindAmmTargetOrdersAccount(market solana.PublicKey) (solana.PublicKey, error) {
	ammId, _, err := solana.FindProgramAddress(
		[][]byte{
			ProgramID.Bytes(),
			market.Bytes(),
			[]byte("target_associated_seed"),
		},
		ProgramID,
	)
	return ammId, err
}

func FindAmmOpenOrdersAccount(market solana.PublicKey) (solana.PublicKey, error) {
	ammId, _, err := solana.FindProgramAddress(
		[][]byte{
			ProgramID.Bytes(),
			market.Bytes(),
			[]byte("open_order_associated_seed"),
		},
		ProgramID,
	)
	return ammId, err
}
