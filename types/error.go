package types

import "errors"

var (
	ErrInvalidPool = errors.New("invalid pool")

	ErrPoolCompleted = errors.New("pool completed")

	ErrNotImplemented = errors.New("not implemented")

	ErrNotFound = errors.New("not found")

	ErrAccountNotInitialized = errors.New("account not initialized")

	ErrInstructionFailed = errors.New("instruction failed")

	ErrTransactionInvalid = errors.New("transaction invalid")

	ErrTransactionFailed = errors.New("transaction failed")

	ErrTxNotLand = errors.New("transaction did not land")

	ErrSlippage = errors.New("slippage error")
)
