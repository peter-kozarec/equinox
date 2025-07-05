package indicators

import (
	"testing"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func Test_NewAtr(t *testing.T) {
	windowSize := 14
	atr := NewAtr(windowSize)

	if atr.windowSize != windowSize {
		t.Errorf("Expected windowSize %d, got %d", windowSize, atr.windowSize)
	}

	if !atr.lastClose.IsZero() {
		t.Error("Expected lastClose to be zero")
	}

	if !atr.lastAtr.IsZero() {
		t.Error("Expected lastAtr to be zero")
	}

	if !atr.currentAtr.IsZero() {
		t.Error("Expected currentAtr to be zero")
	}

	if !atr.currentTr.IsZero() {
		t.Error("Expected currentTr to be zero")
	}

	if atr.Ready() {
		t.Error("Expected ATR to not be ready initially")
	}
}

func TestAtr_FirstBar(t *testing.T) {
	atr := NewAtr(14)

	bar := common.Bar{
		High:  fixed.FromFloat64(100.0),
		Low:   fixed.FromFloat64(95.0),
		Close: fixed.FromFloat64(98.0),
	}

	atr.OnBar(bar)

	if atr.Ready() {
		t.Error("Expected ATR to not be ready after first bar")
	}

	if !atr.currentTr.IsZero() {
		t.Error("Expected currentTr to be zero after first bar")
	}

	if !atr.currentAtr.IsZero() {
		t.Error("Expected currentAtr to be zero after first bar")
	}
}

func TestAtr_MultipleBars(t *testing.T) {
	atr := NewAtr(3)

	bars := []common.Bar{
		{High: fixed.FromFloat64(100.0), Low: fixed.FromFloat64(95.0), Close: fixed.FromFloat64(98.0)},
		{High: fixed.FromFloat64(102.0), Low: fixed.FromFloat64(97.0), Close: fixed.FromFloat64(101.0)},
		{High: fixed.FromFloat64(104.0), Low: fixed.FromFloat64(99.0), Close: fixed.FromFloat64(102.0)},
		{High: fixed.FromFloat64(103.0), Low: fixed.FromFloat64(100.0), Close: fixed.FromFloat64(101.0)},
	}

	for _, bar := range bars {
		atr.OnBar(bar)
	}

	if !atr.Ready() {
		t.Error("Expected ATR to be ready")
	}

	expectedAtr := fixed.FromFloat64(13.0).DivInt(3)
	if !atr.Value().Eq(expectedAtr) {
		t.Errorf("Expected final ATR %v, got %v", expectedAtr, atr.Value())
	}
}

func TestAtr_Reset(t *testing.T) {
	atr := NewAtr(14)

	bars := []common.Bar{
		{High: fixed.FromFloat64(100.0), Low: fixed.FromFloat64(95.0), Close: fixed.FromFloat64(98.0)},
		{High: fixed.FromFloat64(102.0), Low: fixed.FromFloat64(97.0), Close: fixed.FromFloat64(101.0)},
	}

	for _, bar := range bars {
		atr.OnBar(bar)
	}

	if !atr.Ready() {
		t.Error("Expected ATR to be ready before reset")
	}

	atr.Reset()

	if atr.Ready() {
		t.Error("Expected ATR to not be ready after reset")
	}

	if !atr.lastClose.IsZero() {
		t.Error("Expected lastClose to be zero after reset")
	}

	if !atr.lastAtr.IsZero() {
		t.Error("Expected lastAtr to be zero after reset")
	}

	if !atr.currentAtr.IsZero() {
		t.Error("Expected currentAtr to be zero after reset")
	}

	if !atr.currentTr.IsZero() {
		t.Error("Expected currentTr to be zero after reset")
	}
}

func TestAtr_ZeroValues(t *testing.T) {
	atr := NewAtr(14)

	bar := common.Bar{
		High:  fixed.Zero,
		Low:   fixed.Zero,
		Close: fixed.Zero,
	}

	atr.OnBar(bar)

	if atr.Ready() {
		t.Error("Expected ATR to not be ready with zero bar")
	}
}
