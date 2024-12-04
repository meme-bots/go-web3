package common

import "github.com/gagliardetto/solana-go"

const (
	RentBase  = 128
	RentPrice = 6960
	RentATA   = uint64((RentBase + 165) * RentPrice)
)

var (
	JitoRpc = "https://tokyo.mainnet.block-engine.jito.wtf/api/v1/transactions"

	JitoTipPaymentAccounts = []solana.PublicKey{
		solana.MPK("96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5"),
		solana.MPK("HFqU5x63VTqvQss8hp11i4wVV8bD44PvwucfZ2bU7gRe"),
		solana.MPK("Cw8CFyM9FkoMi7K7Crf6HNQqf4uEMzpKw6QNghXLvLkY"),
		solana.MPK("ADaUMid9yfUytqMBgopwjb2DTLSokTSzL1zt6iGPaS49"),
		solana.MPK("DfXygSm4jCyNCybVYYK6DwvWqjKee8pbDmJGcLWNDXjh"),
		solana.MPK("ADuUkR4vqLUMWXxW9gh6D6L8pMSawimctcNZ5pGwDcEt"),
		solana.MPK("DttWaMuVvTiduZRnguLF7jNxTgiMBZ1hyAumKUiL2KRL"),
		solana.MPK("3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT"),
	}
)
