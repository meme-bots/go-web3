// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package multisend

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// MultisendMetaData contains all meta data concerning the Multisend contract.
var MultisendMetaData = &bind.MetaData{
	ABI: "[{\"constant\":false,\"inputs\":[{\"name\":\"_addresses\",\"type\":\"address[]\"},{\"name\":\"_amounts\",\"type\":\"uint256[]\"}],\"name\":\"multiTransfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// MultisendABI is the input ABI used to generate the binding from.
// Deprecated: Use MultisendMetaData.ABI instead.
var MultisendABI = MultisendMetaData.ABI

// Multisend is an auto generated Go binding around an Ethereum contract.
type Multisend struct {
	MultisendCaller     // Read-only binding to the contract
	MultisendTransactor // Write-only binding to the contract
	MultisendFilterer   // Log filterer for contract events
}

// MultisendCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultisendCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultisendTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultisendTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultisendFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultisendFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultisendSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultisendSession struct {
	Contract     *Multisend        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultisendCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultisendCallerSession struct {
	Contract *MultisendCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// MultisendTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultisendTransactorSession struct {
	Contract     *MultisendTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// MultisendRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultisendRaw struct {
	Contract *Multisend // Generic contract binding to access the raw methods on
}

// MultisendCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultisendCallerRaw struct {
	Contract *MultisendCaller // Generic read-only contract binding to access the raw methods on
}

// MultisendTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultisendTransactorRaw struct {
	Contract *MultisendTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultisend creates a new instance of Multisend, bound to a specific deployed contract.
func NewMultisend(address common.Address, backend bind.ContractBackend) (*Multisend, error) {
	contract, err := bindMultisend(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Multisend{MultisendCaller: MultisendCaller{contract: contract}, MultisendTransactor: MultisendTransactor{contract: contract}, MultisendFilterer: MultisendFilterer{contract: contract}}, nil
}

// NewMultisendCaller creates a new read-only instance of Multisend, bound to a specific deployed contract.
func NewMultisendCaller(address common.Address, caller bind.ContractCaller) (*MultisendCaller, error) {
	contract, err := bindMultisend(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultisendCaller{contract: contract}, nil
}

// NewMultisendTransactor creates a new write-only instance of Multisend, bound to a specific deployed contract.
func NewMultisendTransactor(address common.Address, transactor bind.ContractTransactor) (*MultisendTransactor, error) {
	contract, err := bindMultisend(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultisendTransactor{contract: contract}, nil
}

// NewMultisendFilterer creates a new log filterer instance of Multisend, bound to a specific deployed contract.
func NewMultisendFilterer(address common.Address, filterer bind.ContractFilterer) (*MultisendFilterer, error) {
	contract, err := bindMultisend(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultisendFilterer{contract: contract}, nil
}

// bindMultisend binds a generic wrapper to an already deployed contract.
func bindMultisend(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MultisendMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Multisend *MultisendRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Multisend.Contract.MultisendCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Multisend *MultisendRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Multisend.Contract.MultisendTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Multisend *MultisendRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Multisend.Contract.MultisendTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Multisend *MultisendCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Multisend.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Multisend *MultisendTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Multisend.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Multisend *MultisendTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Multisend.Contract.contract.Transact(opts, method, params...)
}

// MultiTransfer is a paid mutator transaction binding the contract method 0x1e89d545.
//
// Solidity: function multiTransfer(address[] _addresses, uint256[] _amounts) payable returns(bool)
func (_Multisend *MultisendTransactor) MultiTransfer(opts *bind.TransactOpts, _addresses []common.Address, _amounts []*big.Int) (*types.Transaction, error) {
	return _Multisend.contract.Transact(opts, "multiTransfer", _addresses, _amounts)
}

// MultiTransfer is a paid mutator transaction binding the contract method 0x1e89d545.
//
// Solidity: function multiTransfer(address[] _addresses, uint256[] _amounts) payable returns(bool)
func (_Multisend *MultisendSession) MultiTransfer(_addresses []common.Address, _amounts []*big.Int) (*types.Transaction, error) {
	return _Multisend.Contract.MultiTransfer(&_Multisend.TransactOpts, _addresses, _amounts)
}

// MultiTransfer is a paid mutator transaction binding the contract method 0x1e89d545.
//
// Solidity: function multiTransfer(address[] _addresses, uint256[] _amounts) payable returns(bool)
func (_Multisend *MultisendTransactorSession) MultiTransfer(_addresses []common.Address, _amounts []*big.Int) (*types.Transaction, error) {
	return _Multisend.Contract.MultiTransfer(&_Multisend.TransactOpts, _addresses, _amounts)
}
