package pumpfun

import (
	"math/big"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go/rpc"
)

func ParseBuyInstruction(tx *rpc.GetTransactionResult, instructionIndex uint16) (solSwaped, tokenSwaped *big.Int) {
	for _, innerInstruction := range tx.Meta.InnerInstructions {
		if innerInstruction.Index == instructionIndex {
			var swapLog SwapLogLayout
			_ = bin.NewBorshDecoder(innerInstruction.Instructions[3].Data[8:]).Decode(&swapLog)
			solSwaped = new(big.Int).Neg(new(big.Int).SetUint64(swapLog.SolAmount))
			tokenSwaped = new(big.Int).SetUint64(swapLog.TokenAmount)
			return
		}
	}
	return
}

func ParseSellInstruction(tx *rpc.GetTransactionResult, instructionIndex uint16) (solSwaped, tokenSwaped *big.Int) {
	for _, innerInstruction := range tx.Meta.InnerInstructions {
		if innerInstruction.Index == instructionIndex {
			var swapLog SwapLogLayout
			_ = bin.NewBorshDecoder(innerInstruction.Instructions[1].Data[8:]).Decode(&swapLog)
			solSwaped = new(big.Int).SetUint64(swapLog.SolAmount)
			tokenSwaped = new(big.Int).Neg(new(big.Int).SetUint64(swapLog.TokenAmount))
			return
		}
	}
	return
}
