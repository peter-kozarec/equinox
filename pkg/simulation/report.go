package simulation

import (
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
	"time"
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

func (report Report) Print(logger *zap.Logger) {
	logger.Info("performance report",
		zap.String("initial_equity", report.InitialEquity.String()),
		zap.String("final_equity", report.FinalEquity.String()),
		zap.String("total_profit", fmt.Sprintf("%s%%", report.TotalProfit.String())),
		zap.String("annualized_return", fmt.Sprintf("%s%%", report.AnnualizedReturn.String())),
		zap.String("max_drawdown", fmt.Sprintf("%s%%", report.MaxDrawdown.String())),
		zap.String("recovery_factor", report.RecoveryFactor.String()),
	)

	logger.Info("trade statistics",
		zap.Int("total_trades", report.TotalTrades),
		zap.Int("winning_trades", report.WinningTrades),
		zap.Int("losing_trades", report.LosingTrades),
		zap.String("win_rate", fmt.Sprintf("%s%%", report.WinRate.String())),
		zap.String("expectancy", report.Expectancy.String()),
		zap.String("profit_factor", report.ProfitFactor.String()),
		zap.String("average_win", report.AverageWin.String()),
		zap.String("average_loss", report.AverageLoss.String()),
		zap.String("risk_reward_ratio", report.RiskRewardRatio.String()),
		zap.String("average_trade_duration", report.AverageTradeDuration.String()),
	)

	logger.Info("risk metrics",
		zap.String("sharpe_ratio", report.SharpeRatio.String()),
		zap.String("sortino_ratio", report.SortinoRatio.String()),
		zap.String("annualized_volatility", fmt.Sprintf("%s%%", report.AnnualizedVolatility.String())),
	)
}
