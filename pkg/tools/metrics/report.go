package metrics

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Report struct {
	StartDate            time.Time
	EndDate              time.Time
	InitialEquity        fixed.Point
	FinalEquity          fixed.Point
	TotalProfit          fixed.Point
	AnnualizedReturn     fixed.Point
	MaxDrawdown          fixed.Point
	TotalTrades          int
	WinningTrades        int
	LosingTrades         int
	WinRate              fixed.Point
	Expectancy           fixed.Point
	ProfitFactor         fixed.Point
	AverageWin           fixed.Point
	AverageLoss          fixed.Point
	RiskRewardRatio      fixed.Point
	AverageTradeDuration time.Duration
	RecoveryFactor       fixed.Point
	SharpeRatio          fixed.Point
	SortinoRatio         fixed.Point
	AnnualizedVolatility fixed.Point
}

func (r Report) Print() {
	slog.Info("trade report",
		"initial_equity", r.InitialEquity,
		"final_equity", r.FinalEquity,
		"total_profit", fmt.Sprintf("%s%%", r.TotalProfit),
		"annualized_return", fmt.Sprintf("%s%%", r.AnnualizedReturn),
		"max_drawdown", fmt.Sprintf("%s%%", r.MaxDrawdown),
		"recovery_factor", r.RecoveryFactor)

	slog.Info("trade statistics",
		"total_trades", r.TotalTrades,
		"winning_trades", r.WinningTrades,
		"losing_trades", r.LosingTrades,
		"win_rate", fmt.Sprintf("%s%%", r.WinRate),
		"expectancy", r.Expectancy,
		"profit_factor", r.ProfitFactor,
		"average_win", r.AverageWin,
		"average_loss", r.AverageLoss,
		"risk_reward_ratio", r.RiskRewardRatio,
		"average_trade_duration", fmt.Sprintf("%.2fm", r.AverageTradeDuration.Minutes()))

	slog.Info("risk metrics",
		"sharpe_ratio", r.SharpeRatio,
		"sortino_ratio", r.SortinoRatio,
		"annualized_volatility", fmt.Sprintf("%s%%", r.AnnualizedVolatility))
}
