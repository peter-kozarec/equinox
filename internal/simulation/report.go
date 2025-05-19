package simulation

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/utility"
	"time"
)

type Report struct {
	StartDate            time.Time
	EndDate              time.Time
	GrossProfit          utility.Fixed
	NetProfit            utility.Fixed
	AnnualizedReturn     utility.Fixed
	MaxDrawdown          utility.Fixed
	TotalTrades          int
	WinningTrades        int
	LosingTrades         int
	WinRate              utility.Fixed
	Expectancy           utility.Fixed
	ProfitFactor         utility.Fixed
	AverageWin           utility.Fixed
	AverageLoss          utility.Fixed
	RiskRewardRatio      utility.Fixed
	AverageTradeDuration utility.Fixed
	RecoveryFactor       utility.Fixed
	SharpeRatio          utility.Fixed
	SortinoRatio         utility.Fixed
	CalmarRatio          utility.Fixed
	AnnualizedVolatility utility.Fixed
}

func (report Report) Print(logger *zap.Logger) {
	logger.Info("performance report",
		zap.String("gross_profit", report.GrossProfit.String()),
		zap.String("net_profit", report.NetProfit.String()),
		zap.String("annualized_return", report.AnnualizedReturn.String()),
		zap.String("max_drawdown", report.MaxDrawdown.String()),
		zap.String("recovery_factor", report.RecoveryFactor.String()),
	)

	logger.Info("trade statistics",
		zap.Int("total_trades", report.TotalTrades),
		zap.Int("winning_trades", report.WinningTrades),
		zap.Int("losing_trades", report.LosingTrades),
		zap.String("win_rate", report.WinRate.String()),
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
		zap.String("calmar_ratio", report.CalmarRatio.String()),
		zap.String("annualized_volatility", report.AnnualizedVolatility.String()),
	)
}
