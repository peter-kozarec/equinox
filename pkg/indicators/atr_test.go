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
		High:  fixed.FromFloat(100.0),
		Low:   fixed.FromFloat(95.0),
		Close: fixed.FromFloat(98.0),
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

func TestAtr_SecondBar(t *testing.T) {
	atr := NewAtr(14)

	// First bar
	bar1 := common.Bar{
		High:  fixed.FromFloat(100.0),
		Low:   fixed.FromFloat(95.0),
		Close: fixed.FromFloat(98.0),
	}
	atr.OnBar(bar1)

	bar2 := common.Bar{
		High:  fixed.FromFloat(102.0),
		Low:   fixed.FromFloat(97.0),
		Close: fixed.FromFloat(101.0),
	}
	atr.OnBar(bar2)

	if !atr.Ready() {
		t.Error("Expected ATR to be ready after second bar")
	}

	expectedTr := fixed.FromFloat(5.0)
	if !atr.TrueRange().Eq(expectedTr) {
		t.Errorf("Expected TR %v, got %v", expectedTr, atr.TrueRange())
	}

	if !atr.AverageTrueRange().Eq(expectedTr) {
		t.Errorf("Expected ATR %v, got %v", expectedTr, atr.AverageTrueRange())
	}
}

func TestAtr_MultipleBars(t *testing.T) {
	atr := NewAtr(3)

	bars := []common.Bar{
		{High: fixed.FromFloat(100.0), Low: fixed.FromFloat(95.0), Close: fixed.FromFloat(98.0)},
		{High: fixed.FromFloat(102.0), Low: fixed.FromFloat(97.0), Close: fixed.FromFloat(101.0)},
		{High: fixed.FromFloat(104.0), Low: fixed.FromFloat(99.0), Close: fixed.FromFloat(102.0)},
		{High: fixed.FromFloat(103.0), Low: fixed.FromFloat(100.0), Close: fixed.FromFloat(101.0)},
	}

	for _, bar := range bars {
		atr.OnBar(bar)
	}

	if !atr.Ready() {
		t.Error("Expected ATR to be ready")
	}

	expectedTr := fixed.FromFloat(3.0)
	if !atr.TrueRange().Eq(expectedTr) {
		t.Errorf("Expected final TR %v, got %v", expectedTr, atr.TrueRange())
	}

	expectedAtr := fixed.FromFloat(13.0).DivInt(3)
	if !atr.AverageTrueRange().Eq(expectedAtr) {
		t.Errorf("Expected final ATR %v, got %v", expectedAtr, atr.AverageTrueRange())
	}
}

func TestAtr_TrueRangeCalculation(t *testing.T) {
	atr := NewAtr(14)

	// First bar
	bar1 := common.Bar{
		High:  fixed.FromFloat(50.0),
		Low:   fixed.FromFloat(45.0),
		Close: fixed.FromFloat(48.0),
	}
	atr.OnBar(bar1)

	testCases := []struct {
		name       string
		bar        common.Bar
		expectedTr fixed.Point
	}{
		{
			name: "High-Low is maximum",
			bar: common.Bar{
				High:  fixed.FromFloat(55.0),
				Low:   fixed.FromFloat(47.0),
				Close: fixed.FromFloat(52.0),
			},
			expectedTr: fixed.FromFloat(8.0),
		},
		{
			name: "High-PrevClose is maximum",
			bar: common.Bar{
				High:  fixed.FromFloat(60.0),
				Low:   fixed.FromFloat(55.0),
				Close: fixed.FromFloat(58.0),
			},
			expectedTr: fixed.FromFloat(8.0),
		},
		{
			name: "Low-PrevClose is maximum",
			bar: common.Bar{
				High:  fixed.FromFloat(62.0),
				Low:   fixed.FromFloat(50.0),
				Close: fixed.FromFloat(55.0),
			},
			expectedTr: fixed.FromFloat(12.0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			atr.OnBar(tc.bar)
			if !atr.TrueRange().Eq(tc.expectedTr) {
				t.Errorf("Expected TR %v, got %v", tc.expectedTr, atr.TrueRange())
			}
		})
	}
}

func TestAtr_Reset(t *testing.T) {
	atr := NewAtr(14)

	bars := []common.Bar{
		{High: fixed.FromFloat(100.0), Low: fixed.FromFloat(95.0), Close: fixed.FromFloat(98.0)},
		{High: fixed.FromFloat(102.0), Low: fixed.FromFloat(97.0), Close: fixed.FromFloat(101.0)},
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
