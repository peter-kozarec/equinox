package common

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type Instrument struct {
	Symbol           string      `json:"symbol"`
	Id               int64       `json:"id"`
	Digits           int         `json:"digits"`
	DenominationUnit string      `json:"denomination_unit,omitempty"`
	ContractSize     fixed.Point `json:"contract_size"`
	PipSize          fixed.Point `json:"pip_size,omitempty"`
	CommissionPerLot fixed.Point `json:"commission_per_lot,omitempty"`
	PipSlippage      fixed.Point `json:"pip_slippage,omitempty"`
}
