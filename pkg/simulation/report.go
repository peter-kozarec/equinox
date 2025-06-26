package simulation

import (
	"fmt"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
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

func (r Report) Print(logger *zap.Logger) {
	logger.Info("trade report",
		zap.String("initial_equity", r.InitialEquity.String()),
		zap.String("final_equity", r.FinalEquity.String()),
		zap.String("total_profit", fmt.Sprintf("%s%%", r.TotalProfit.String())),
		zap.String("annualized_return", fmt.Sprintf("%s%%", r.AnnualizedReturn.String())),
		zap.String("max_drawdown", fmt.Sprintf("%s%%", r.MaxDrawdown.String())),
		zap.String("recovery_factor", r.RecoveryFactor.String()),
	)

	logger.Info("trade statistics",
		zap.Int("total_trades", r.TotalTrades),
		zap.Int("winning_trades", r.WinningTrades),
		zap.Int("losing_trades", r.LosingTrades),
		zap.String("win_rate", fmt.Sprintf("%s%%", r.WinRate.String())),
		zap.String("expectancy", r.Expectancy.String()),
		zap.String("profit_factor", r.ProfitFactor.String()),
		zap.String("average_win", r.AverageWin.String()),
		zap.String("average_loss", r.AverageLoss.String()),
		zap.String("risk_reward_ratio", r.RiskRewardRatio.String()),
		zap.String("average_trade_duration", r.AverageTradeDuration.String()),
	)

	logger.Info("risk metrics",
		zap.String("sharpe_ratio", r.SharpeRatio.String()),
		zap.String("sortino_ratio", r.SortinoRatio.String()),
		zap.String("annualized_volatility", fmt.Sprintf("%s%%", r.AnnualizedVolatility.String())),
	)
}
