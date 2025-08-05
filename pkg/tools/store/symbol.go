package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	ErrSymbolNotPresent = errors.New("symbol is not present in symbol table")
)

type SymbolStore struct {
	symbols []exchange.SymbolInfo
}

func CreateSymbolStore(symbols ...exchange.SymbolInfo) SymbolStore {
	return SymbolStore{
		symbols: symbols,
	}
}

func (s SymbolStore) Contains(symbolName string) bool {
	if _, err := s.Get(symbolName); err != nil {
		return false
	}
	return true
}

func (s SymbolStore) Get(symbolName string) (exchange.SymbolInfo, error) {
	for _, symbol := range s.symbols {
		if strings.EqualFold(symbol.SymbolName, symbolName) {
			return symbol, nil
		}
	}
	return exchange.SymbolInfo{}, fmt.Errorf("unable to get symbol with name %s: %w", symbolName, ErrSymbolNotPresent)
}

func (s SymbolStore) MustGet(symbolName string) exchange.SymbolInfo {
	symbol, err := s.Get(symbolName)
	if err != nil {
		panic(err.Error())
	}
	return symbol
}

func CreateSymbolTestStore() SymbolStore {
	return CreateSymbolStore([]exchange.SymbolInfo{
		{
			SymbolName:    "EURUSD",
			QuoteCurrency: "USD",
			Class:         exchange.Forex,
			Digits:        5,
			PipSize:       fixed.FromFloat64(0.0001),
			ContractSize:  fixed.FromFloat64(100_000),
			Leverage:      fixed.FromFloat64(30),
		},
	}...)
}
