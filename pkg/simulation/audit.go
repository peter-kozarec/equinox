package simulation

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type accountSnapshot struct {
	balance fixed.Point
	equity  fixed.Point
	t       time.Time
}

type Audit struct {
	minSnapshotInterval time.Duration

	accountSnapshots []accountSnapshot
	closedPositions  []common.Position
}

func NewAudit(minSnapshotInterval time.Duration) *Audit {
	return &Audit{
		minSnapshotInterval: minSnapshotInterval,
	}
}

func (a *Audit) AddAccountSnapshot(balance, equity fixed.Point, t time.Time) {
	if len(a.accountSnapshots) == 0 ||
		t.Sub(a.accountSnapshots[len(a.accountSnapshots)-1].t) >= a.minSnapshotInterval {
		a.addSnapshot(balance, equity, t)
	}
}

func (a *Audit) AddClosedPosition(position common.Position) {
	a.closedPositions = append(a.closedPositions, position)
}

func (a *Audit) GenerateReport() Report {

	report := Report{}

	auditedDays := a.dayCount()
	year := fixed.FromInt64(36500, 2)

	report.InitialEquity = a.accountSnapshots[0].equity
	report.StartDate = a.accountSnapshots[0].t
	report.FinalEquity = a.accountSnapshots[len(a.accountSnapshots)-1].equity
	report.EndDate = a.accountSnapshots[len(a.accountSnapshots)-1].t

	// --- Return Metrics ---
	report.TotalProfit = report.FinalEquity.Div(report.InitialEquity).Sub(fixed.One).MulInt64(100).Rescale(2)
	if auditedDays > 0 && report.InitialEquity.Gt(fixed.Zero) && report.FinalEquity.Gt(fixed.Zero) {
		ratio := report.FinalEquity.Div(report.InitialEquity)
		exponent := year.DivInt64(int64(auditedDays))
		report.AnnualizedReturn = ratio.Pow(exponent).Sub(fixed.One).MulInt64(100).Rescale(2)
	} else {
		report.AnnualizedReturn = fixed.Zero // or some error/NaN marker
	}

	// --- Max Drawdown ---
	maxEquity := report.InitialEquity
	for _, snapshot := range a.accountSnapshots {
		if snapshot.equity.Gt(maxEquity) {
			maxEquity = snapshot.equity
		}
		drawdown := maxEquity.Sub(snapshot.equity).Div(maxEquity)
		if drawdown.Gt(report.MaxDrawdown) {
			report.MaxDrawdown = drawdown
		}
	}

	// --- Trade Statistics ---
	var (
		totalDuration time.Duration
		totalProfit   fixed.Point
		totalLoss     fixed.Point
	)
	for _, position := range a.closedPositions {
		report.TotalTrades++

		// Calc duration
		if !position.OpenTime.IsZero() && !position.CloseTime.IsZero() && position.CloseTime.After(position.OpenTime) {
			totalDuration += position.CloseTime.Sub(position.OpenTime)
		}

		// Aggregate profit
		if position.NetProfit.Gt(fixed.Zero) {
			totalProfit = totalProfit.Add(position.NetProfit)
			report.WinningTrades++
		} else if position.NetProfit.Lte(fixed.Zero) {
			totalLoss = totalLoss.Add(position.NetProfit.Neg())
			report.LosingTrades++
		}
	}

	// --- Averages & Ratios ---
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

	// --- Risk Metrics: Volatility, Sharpe, Sortino ---
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

func (a *Audit) addSnapshot(balance, equity fixed.Point, t time.Time) {
	a.accountSnapshots = append(a.accountSnapshots, accountSnapshot{
		balance: balance,
		equity:  equity,
		t:       t,
	})
}

func (a *Audit) dayCount() int {
	if len(a.accountSnapshots) < 2 {
		return 1
	}
	start := a.accountSnapshots[0].t
	end := a.accountSnapshots[len(a.accountSnapshots)-1].t
	return int(end.Sub(start).Hours()/24) + 1
}

func (a *Audit) dailyReturns() []fixed.Point {
	var dailyReturns []fixed.Point
	if len(a.accountSnapshots) < 2 {
		return dailyReturns
	}

	var (
		prevDate   = a.accountSnapshots[0].t.Truncate(24 * time.Hour)
		prevEquity = a.accountSnapshots[0].equity
	)

	for _, snapshot := range a.accountSnapshots[1:] {
		currDate := snapshot.t.Truncate(24 * time.Hour)

		if currDate.After(prevDate) {
			ret := snapshot.equity.Div(prevEquity).Sub(fixed.One)
			dailyReturns = append(dailyReturns, ret)

			prevDate = currDate
			prevEquity = snapshot.equity
		}
	}

	return dailyReturns
}
