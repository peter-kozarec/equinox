package metrics

import (
	"context"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	equitySnapshotInterval = time.Minute
)

type Audit struct {
	equities  []common.Equity
	positions []common.Position
}

func NewAudit() *Audit {
	return &Audit{}
}

func (a *Audit) OnEquity(_ context.Context, equity common.Equity) {
	if len(a.equities) == 0 || equity.TimeStamp.Sub(a.equities[len(a.equities)-1].TimeStamp) >= equitySnapshotInterval {
		a.equities = append(a.equities, equity)
	}
}

func (a *Audit) OnPositionClosed(_ context.Context, position common.Position) {
	a.positions = append(a.positions, position)
}

func (a *Audit) GenerateReport() Report {
	report := Report{}

	auditedDays := a.dayCount()
	year := fixed.FromInt64(36500, 2)

	report.InitialEquity = a.equities[0].Value
	report.StartDate = a.equities[0].TimeStamp
	report.FinalEquity = a.equities[len(a.equities)-1].Value
	report.EndDate = a.equities[len(a.equities)-1].TimeStamp

	report.TotalProfit = report.FinalEquity.Div(report.InitialEquity).Sub(fixed.One).MulInt64(100).Rescale(2)
	if auditedDays > 0 && report.InitialEquity.Gt(fixed.Zero) && report.FinalEquity.Gt(fixed.Zero) {
		ratio := report.FinalEquity.Div(report.InitialEquity)
		exponent := year.DivInt64(int64(auditedDays))
		report.AnnualizedReturn = ratio.Pow(exponent).Sub(fixed.One).MulInt64(100).Rescale(2)
	} else {
		report.AnnualizedReturn = fixed.Zero
	}

	maxEquity := report.InitialEquity
	for _, eq := range a.equities {
		if eq.Value.Gt(maxEquity) {
			maxEquity = eq.Value
		}
		drawdown := maxEquity.Sub(eq.Value).Div(maxEquity)
		if drawdown.Gt(report.MaxDrawdown) {
			report.MaxDrawdown = drawdown
		}
	}

	var (
		totalDuration time.Duration
		totalProfit   fixed.Point
		totalLoss     fixed.Point
	)
	for _, position := range a.positions {
		report.TotalTrades++

		if !position.OpenTime.IsZero() && !position.CloseTime.IsZero() && position.CloseTime.After(position.OpenTime) {
			totalDuration += position.CloseTime.Sub(position.OpenTime)
		}

		if position.NetProfit.Gt(fixed.Zero) {
			totalProfit = totalProfit.Add(position.NetProfit)
			report.WinningTrades++
		} else if position.NetProfit.Lte(fixed.Zero) {
			totalLoss = totalLoss.Add(position.NetProfit.Neg())
			report.LosingTrades++
		}
	}

	if report.WinningTrades > 0 {
		report.AverageWin = totalProfit.DivInt64(int64(report.WinningTrades))
	}
	if report.LosingTrades > 0 {
		report.AverageLoss = totalLoss.DivInt64(int64(report.LosingTrades))
	}
	if totalLoss.Gt(fixed.Zero) {
		report.ProfitFactor = totalProfit.Div(totalLoss)
	}
	if report.AverageLoss.Gt(fixed.Zero) {
		report.RiskRewardRatio = report.AverageWin.Div(report.AverageLoss)
	}
	if report.TotalTrades > 0 {
		report.Expectancy = totalProfit.Sub(totalLoss).DivInt64(int64(report.TotalTrades))
		report.AverageTradeDuration = totalDuration / time.Duration(report.TotalTrades)
		report.WinRate = fixed.FromInt64(int64(report.WinningTrades), 0).DivInt64(int64(report.TotalTrades)).MulInt64(100).Rescale(2)
	}
	if report.MaxDrawdown.Gt(fixed.Zero) {
		report.RecoveryFactor = report.TotalProfit.Div(report.MaxDrawdown)
	}
	report.MaxDrawdown = report.MaxDrawdown.MulInt64(100).Rescale(2)

	dailyReturns := a.dailyReturns()
	meanReturn := fixed.Mean(dailyReturns)
	vol := fixed.StdDev(dailyReturns, meanReturn)

	if !meanReturn.IsZero() && !vol.IsZero() {
		report.AnnualizedVolatility = vol.Mul(fixed.Sqrt252).MulInt64(100).Rescale(2)
		report.SharpeRatio = fixed.SharpeRatio(dailyReturns, fixed.Zero).Mul(fixed.Sqrt252).Rescale(5)
		report.SortinoRatio = fixed.SortinoRatio(dailyReturns, fixed.Zero).Mul(fixed.Sqrt252).Rescale(5)
	}

	return report
}

func (a *Audit) dayCount() int {
	if len(a.equities) < 2 {
		return 1
	}
	start := a.equities[0].TimeStamp
	end := a.equities[len(a.equities)-1].TimeStamp
	return int(end.Sub(start).Hours()/24) + 1
}

func (a *Audit) dailyReturns() []fixed.Point {
	var dailyReturns []fixed.Point
	if len(a.equities) < 2 {
		return dailyReturns
	}

	var (
		prevDate   = a.equities[0].TimeStamp.Truncate(24 * time.Hour)
		prevEquity = a.equities[0].Value
	)

	for _, eq := range a.equities[1:] {
		currDate := eq.TimeStamp.Truncate(24 * time.Hour)

		if currDate.After(prevDate) {
			ret := eq.Value.Div(prevEquity).Sub(fixed.One)
			dailyReturns = append(dailyReturns, ret)

			prevDate = currDate
			prevEquity = eq.Value
		}
	}

	return dailyReturns
}
