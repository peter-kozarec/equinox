package sandbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	positionStatusPendingOpen  common.PositionStatus = "pending-open"
	positionStatusPendingClose common.PositionStatus = "pending-close"

	simulatorComponentName = "exchange.sandbox.simulator"
)

var (
	minMaintenanceMarginRate = fixed.Five

	ErrRouterIsNil         = errors.New("router is nil")
	ErrAccCurrencyNotSet   = errors.New("account currency not set")
	ErrStartBalanceInvalid = errors.New("start balance is invalid")
	ErrSymbolMapIsEmpty    = errors.New("symbol map is empty")
	ErrInvalidLeverage     = errors.New("invalid leverage")
)

type Simulator struct {
	router          *bus.Router
	accountCurrency string

	rateProvider          exchange.RateProvider
	symbolsMap            map[string]exchange.SymbolInfo
	commissionHandler     CommissionHandler
	swapHandler           SwapHandler
	slippageHandler       SlippageHandler
	maintenanceMarginRate fixed.Point

	firstPostDone bool
	equity        fixed.Point
	balance       fixed.Point
	freeMargin    fixed.Point

	simulationTime time.Time
	lastTickMap    map[string]common.Tick

	positionIdCounter common.PositionId
	openPositions     []*common.Position
	openOrders        []*common.Order
}

func NewSimulator(router *bus.Router, accountCurrency string, startBalance fixed.Point, options ...Option) (*Simulator, error) {
	if router == nil {
		return nil, ErrRouterIsNil
	}
	if accountCurrency == "" {
		return nil, ErrAccCurrencyNotSet
	}
	if startBalance.Lte(fixed.Zero) {
		return nil, ErrStartBalanceInvalid
	}

	s := &Simulator{
		router:                router,
		accountCurrency:       accountCurrency,
		symbolsMap:            make(map[string]exchange.SymbolInfo),
		maintenanceMarginRate: minMaintenanceMarginRate,
		equity:                startBalance,
		balance:               startBalance,
		freeMargin:            startBalance,
		lastTickMap:           make(map[string]common.Tick),
	}

	for _, option := range options {
		option(s)
	}

	if len(s.symbolsMap) == 0 {
		return nil, ErrSymbolMapIsEmpty
	}

	for k, v := range s.symbolsMap {
		if v.Leverage.IsZero() {
			return nil, fmt.Errorf("invalid symbol info for %s: %w", k, ErrInvalidLeverage)
		}
	}

	return s, nil
}

func (s *Simulator) OnOrder(_ context.Context, order common.Order) {
	if err := s.validateOrder(order); err != nil {
		s.postOrderRejected(order, fmt.Sprintf("order with trace id %d validation failed: %s", order.TraceID, err.Error()))
	} else {
		orderAccepted := common.OrderAccepted{
			OriginalOrder: order,
			Source:        simulatorComponentName,
			ExecutionId:   utility.GetExecutionID(),
			TraceID:       utility.CreateTraceID(),
			TimeStamp:     s.simulationTime,
		}
		if err := s.router.Post(bus.OrderAcceptanceEvent, orderAccepted); err != nil {
			slog.Error("unable to post order accepted event, dropping order...",
				"error", err, "order_accepted", orderAccepted)
			return
		}

		orderCopy := order
		s.openOrders = append(s.openOrders, &orderCopy)
	}
}

func (s *Simulator) OnTick(_ context.Context, tick common.Tick) {
	if err := s.validateTick(tick); err != nil {
		slog.Error("unable to validate tick, dropping tick...",
			"error", err, "tick", tick)
		return
	}

	s.simulationTime = tick.TimeStamp
	s.lastTickMap[strings.ToUpper(tick.Symbol)] = tick

	if !s.firstPostDone {
		s.firstPostDone = true
		s.postBalance()
		s.postEquity()
	}

	lastBalance := s.balance
	lastEquity := s.equity

	s.checkOrders(tick)
	s.checkPositions(tick)
	s.processPendingChanges(tick)
	s.checkMargin(tick)

	if !lastBalance.Eq(s.balance) {
		s.postBalance()
	}
	if !lastEquity.Eq(s.equity) {
		s.postEquity()
	}
}

func (s *Simulator) CloseAllOpenPositions() {
	s.equity = s.balance

	for _, position := range s.openPositions {
		tick, ok := s.lastTickMap[strings.ToUpper(position.Symbol)]
		if !ok {
			slog.Warn("no tick for symbol, skipping close",
				"position", position)
			continue
		}

		closePrice := tick.Bid
		if position.Side == common.PositionSideShort {
			closePrice = tick.Ask
		}

		s.calcPositionProfits(position, closePrice)
		s.equity = s.equity.Add(position.NetProfit)

		position.Status = common.PositionStatusClosed
		position.ClosePrice = closePrice
		position.CloseTime = s.simulationTime

		positionCopy := *position
		if err := s.router.Post(bus.PositionCloseEvent, positionCopy); err != nil {
			slog.Warn("unable to post position closed event",
				"error", err, "position", positionCopy)
		}
	}

	s.balance = s.equity
	s.openPositions = nil
}

func (s *Simulator) checkPositions(tick common.Tick) {
	for _, position := range s.openPositions {
		if s.shouldClosePosition(*position, tick) {
			position.Status = positionStatusPendingClose
		}
	}
}

func (s *Simulator) checkOrders(tick common.Tick) {
	tmpOpenOrders := make([]*common.Order, 0, len(s.openOrders))

	for _, order := range s.openOrders {
		if !strings.EqualFold(order.Symbol, tick.Symbol) {
			tmpOpenOrders = append(tmpOpenOrders, order)
			continue
		}

		switch order.Command {
		case common.OrderCommandPositionOpen:
			switch order.Type {
			case common.OrderTypeMarket:
				position, err := s.executeOpenOrder(*order, tick)
				if err != nil {
					s.postOrderRejected(*order, fmt.Sprintf("market execution failed: %v", err))
					continue
				}

				order.FilledSize = order.FilledSize.Add(position.Size)
				remaining := order.Size.Sub(order.FilledSize)
				s.openPositions = append(s.openPositions, position)

				switch order.TimeInForce {
				case common.TimeInForceImmediateOrCancel:
					if order.FilledSize.Gt(fixed.Zero) {
						s.postOrderFilled(*order, position.Id)
					}
					if remaining.Gt(fixed.Zero) {
						s.postOrderCancel(*order, remaining)
					}
				case common.TimeInForceFillOrKill:
					if !order.FilledSize.Eq(order.Size) {
						s.openPositions = s.openPositions[:len(s.openPositions)-1]
						s.postOrderCancel(*order, order.Size)
					} else {
						s.postOrderFilled(*order, position.Id)
					}
				default:
					s.postOrderFilled(*order, position.Id)
					if remaining.Gt(fixed.Zero) {
						tmpOpenOrders = append(tmpOpenOrders, order)
					}
				}
			case common.OrderTypeLimit:
				if !s.shouldExecuteLimitOrder(*order, tick) {
					if order.TimeInForce == common.TimeInForceImmediateOrCancel || order.TimeInForce == common.TimeInForceFillOrKill ||
						(order.TimeInForce == common.TimeInForceGoodTillDate && s.simulationTime.After(order.ExpireTime)) {
						s.postOrderCancel(*order, order.Size.Sub(order.FilledSize))
					} else {
						tmpOpenOrders = append(tmpOpenOrders, order)
					}
					continue
				}

				position, err := s.executeOpenOrder(*order, tick)
				if err != nil {
					s.postOrderRejected(*order, fmt.Sprintf("limit execution failed: %v", err))
					continue
				}

				order.FilledSize = order.FilledSize.Add(position.Size)
				remaining := order.Size.Sub(order.FilledSize)
				s.openPositions = append(s.openPositions, position)

				switch order.TimeInForce {
				case common.TimeInForceImmediateOrCancel:
					if order.FilledSize.Gt(fixed.Zero) {
						s.postOrderFilled(*order, position.Id)
					}
					if remaining.Gt(fixed.Zero) {
						s.postOrderCancel(*order, remaining)
					}
				case common.TimeInForceFillOrKill:
					if !order.FilledSize.Eq(order.Size) {
						s.openPositions = s.openPositions[:len(s.openPositions)-1]
						s.postOrderCancel(*order, order.Size)
					} else {
						s.postOrderFilled(*order, position.Id)
					}
				case common.TimeInForceGoodTillDate:
					if s.simulationTime.After(order.ExpireTime) {
						s.openPositions = s.openPositions[:len(s.openPositions)-1]
						s.postOrderCancel(*order, remaining)
					} else if remaining.Gt(fixed.Zero) {
						tmpOpenOrders = append(tmpOpenOrders, order)
					} else {
						s.postOrderFilled(*order, position.Id)
					}
				case common.TimeInForceGoodTillCancel:
					if remaining.Gt(fixed.Zero) {
						tmpOpenOrders = append(tmpOpenOrders, order)
					} else {
						s.postOrderFilled(*order, position.Id)
					}
				}
			}
		case common.OrderCommandPositionClose:
			switch order.Type {
			case common.OrderTypeMarket:
				position, filledSize, err := s.executeCloseOrder(*order, tick)
				if err != nil {
					s.postOrderRejected(*order, fmt.Sprintf("market close failed: %v", err))
					continue
				}

				order.FilledSize = order.FilledSize.Add(filledSize)
				remaining := order.Size.Sub(order.FilledSize)

				if order.FilledSize.Gt(fixed.Zero) {
					if !order.FilledSize.Eq(position.Size) {
						newPosition := *position
						newPosition.Size = position.Size.Sub(order.FilledSize)
						newPosition.Status = common.PositionStatusOpen
						s.openPositions = append(s.openPositions, &newPosition)

						position.Size = filledSize
					}

					s.postOrderFilled(*order, position.Id)
				}

				switch order.TimeInForce {
				case common.TimeInForceImmediateOrCancel:
					if remaining.Gt(fixed.Zero) {
						s.postOrderCancel(*order, remaining)
					}
				case common.TimeInForceFillOrKill:
					if !order.FilledSize.Eq(order.Size) {
						position.Status = common.PositionStatusOpen
						s.postOrderCancel(*order, order.Size)
					}
				default:
					if remaining.Gt(fixed.Zero) {
						tmpOpenOrders = append(tmpOpenOrders, order)
					}
				}
			case common.OrderTypeLimit:
				if !s.shouldExecuteLimitOrder(*order, tick) {
					if order.TimeInForce == common.TimeInForceImmediateOrCancel ||
						order.TimeInForce == common.TimeInForceFillOrKill {
						s.postOrderCancel(*order, order.Size.Sub(order.FilledSize))
					} else {
						tmpOpenOrders = append(tmpOpenOrders, order)
					}
					continue
				}

				position, filledSize, err := s.executeCloseOrder(*order, tick)
				if err != nil {
					s.postOrderRejected(*order, fmt.Sprintf("limit close failed: %v", err))
					continue
				}

				order.FilledSize = order.FilledSize.Add(filledSize)
				remaining := order.Size.Sub(order.FilledSize)

				switch order.TimeInForce {
				case common.TimeInForceImmediateOrCancel:
					if order.FilledSize.Gt(fixed.Zero) {
						if remaining.Gt(fixed.Zero) {
							newPosition := *position
							newPosition.Size = remaining
							newPosition.Status = common.PositionStatusOpen
							s.openPositions = append(s.openPositions, &newPosition)

							position.Size = filledSize
						}
						s.postOrderFilled(*order, position.Id)
					}
					if remaining.Gt(fixed.Zero) {
						s.postOrderCancel(*order, remaining)
					}
				case common.TimeInForceFillOrKill:
					if !order.FilledSize.Eq(order.Size) {
						position.Status = common.PositionStatusOpen
						s.postOrderCancel(*order, order.Size)
					} else {
						position.Status = positionStatusPendingClose
						s.postOrderFilled(*order, position.Id)
					}
				case common.TimeInForceGoodTillDate:
					if s.simulationTime.After(order.ExpireTime) {
						s.postOrderCancel(*order, remaining)
					} else if remaining.Gt(fixed.Zero) {
						if order.FilledSize.Gt(fixed.Zero) {
							newPosition := *position
							newPosition.Size = remaining
							newPosition.Status = common.PositionStatusOpen
							s.openPositions = append(s.openPositions, &newPosition)

							position.Size = filledSize
							s.postOrderFilled(*order, position.Id)
						}
						tmpOpenOrders = append(tmpOpenOrders, order)
					} else {
						position.Status = positionStatusPendingClose
						s.postOrderFilled(*order, position.Id)
					}
				case common.TimeInForceGoodTillCancel:
					if remaining.Gt(fixed.Zero) {
						if order.FilledSize.Gt(fixed.Zero) {
							newPosition := *position
							newPosition.Size = remaining
							newPosition.Status = common.PositionStatusOpen
							s.openPositions = append(s.openPositions, &newPosition)

							position.Size = filledSize
							s.postOrderFilled(*order, position.Id)
						}
						tmpOpenOrders = append(tmpOpenOrders, order)
					} else {
						position.Status = positionStatusPendingClose
						s.postOrderFilled(*order, position.Id)
					}
				}
			}
		case common.OrderCommandPositionModify:
			if err := s.modifyPosition(*order, tick); err != nil {
				s.postOrderRejected(*order, fmt.Sprintf("position modification failed: %v", err))
			}
		default:
			slog.Error("unknown command", "cmd", order.Command)
		}
	}

	s.openOrders = tmpOpenOrders
}

func (s *Simulator) checkMargin(tick common.Tick) {
	s.calcFreeMargin()
	if s.equity.IsZero() {
		s.equity = fixed.FromFloat64(0.0001)
	}
	freeMarginRate := s.freeMargin.Div(s.equity).MulInt(100)
	if freeMarginRate.Lte(s.maintenanceMarginRate) {
		if len(s.openPositions) == 0 {
			slog.Error("no open positions to close",
				"free_margin_rate", freeMarginRate,
				"maintenance_margin_rate", s.maintenanceMarginRate)
			return
		}
		positionToClose := s.openPositions[0]
		s.openPositions = s.openPositions[1:]

		tmpPosition := *positionToClose
		s.equity = s.equity.Sub(tmpPosition.NetProfit)

		closePrice := tick.Ask
		if positionToClose.Side == common.PositionSideLong {
			closePrice = tick.Bid
		}

		positionToClose.ClosePrice = closePrice
		positionToClose.CloseTime = s.simulationTime
		positionToClose.Status = common.PositionStatusClosed
		positionToClose.TimeStamp = s.simulationTime
		s.calcPositionProfits(positionToClose, closePrice)

		if err := s.router.Post(bus.PositionCloseEvent, *positionToClose); err != nil {
			slog.Warn("unable to post position closed event",
				"error", err,
				"position", tmpPosition)

			s.equity = s.equity.Add(tmpPosition.NetProfit)
			s.openPositions = append(s.openPositions, &tmpPosition)
			return
		}
		s.equity = s.equity.Add(positionToClose.NetProfit)
		s.balance = s.balance.Add(positionToClose.NetProfit)
		s.checkMargin(tick)
	}
}

func (s *Simulator) processPendingChanges(tick common.Tick) {
	tmpOpenPositions := make([]*common.Position, 0, len(s.openPositions))
	s.equity = s.balance

	for _, position := range s.openPositions {
		if !strings.EqualFold(position.Symbol, tick.Symbol) {
			tmpOpenPositions = append(tmpOpenPositions, position)
			continue
		}

		openPrice := tick.Bid
		closePrice := tick.Ask
		if position.Side == common.PositionSideLong {
			openPrice = tick.Ask
			closePrice = tick.Bid
		}

		switch position.Status {
		case positionStatusPendingOpen:
			position.Status = common.PositionStatusOpen
			position.OpenPrice = openPrice
			position.OpenTime = tick.TimeStamp
			if s.slippageHandler != nil {
				position.Slippage = s.slippageHandler(*position)
			}
			if err := s.router.Post(bus.PositionOpenEvent, *position); err != nil {
				slog.Warn("unable to post position opened event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		case positionStatusPendingClose:
			position.Status = common.PositionStatusClosed
			position.ClosePrice = closePrice
			position.CloseTime = tick.TimeStamp
			if s.slippageHandler != nil {
				position.Slippage = position.Slippage.Add(s.slippageHandler(*position))
			}
			s.calcPositionProfits(position, closePrice)
			position.TimeStamp = s.simulationTime
			s.balance = s.balance.Add(position.NetProfit)
			if err := s.router.Post(bus.PositionCloseEvent, *position); err != nil {
				slog.Warn("unable to post position closed event", "error", err)
			}
		default:
			s.calcPositionProfits(position, closePrice)
			position.TimeStamp = s.simulationTime
			s.equity = s.equity.Add(position.NetProfit)
			if err := s.router.Post(bus.PositionUpdateEvent, *position); err != nil {
				slog.Warn("unable to post position pnl updated event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		}
	}

	s.openPositions = tmpOpenPositions
}

func (s *Simulator) executeCloseOrder(order common.Order, tick common.Tick) (*common.Position, fixed.Point, error) {
	// ToDo: Liquidity is not taken from the volume when position is opened, so if there are multiple orders that
	//       exceeds the available volume together, they will all get filled even so they should not
	for _, position := range s.openPositions {
		if position.Id == order.PositionId {
			availableLiquidity := tick.BidVolume
			if order.Side == common.OrderSideBuy {
				availableLiquidity = tick.AskVolume
			}

			size := order.Size.Sub(order.FilledSize)
			if availableLiquidity.IsZero() {
				return nil, fixed.Zero, fmt.Errorf("available liquidity is zero")
			}

			if size.Gt(availableLiquidity) {
				size = availableLiquidity
				size = size.Rescale(2)
			}
			position.Status = positionStatusPendingClose
			return position, size, nil
		}
	}
	return nil, fixed.Zero, fmt.Errorf("position with id %d not found", order.PositionId)
}

func (s *Simulator) executeOpenOrder(order common.Order, tick common.Tick) (*common.Position, error) {
	// ToDo: Liquidity is not taken from the volume when position is opened, so if there are multiple orders that
	//       exceeds the available volume together, they will all get filled even so they should not
	var availableLiquidity fixed.Point
	var positionSide common.PositionSide

	if order.Side == common.OrderSideBuy {
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Gte(order.TakeProfit) {
			return nil, fmt.Errorf("stop loss must be less than take profit")
		}
		if !order.StopLoss.IsZero() && order.StopLoss.Gt(tick.Bid) {
			return nil, fmt.Errorf("stop loss must be less than bid")
		}
		if !order.TakeProfit.IsZero() && order.TakeProfit.Lt(tick.Bid) {
			return nil, fmt.Errorf("take profit must be greater than bid")
		}
		positionSide = common.PositionSideLong
		availableLiquidity = tick.AskVolume
	} else {
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Lte(order.TakeProfit) {
			return nil, fmt.Errorf("stop loss must be greater than take profit")
		}
		if !order.StopLoss.IsZero() && order.StopLoss.Lt(tick.Ask) {
			return nil, fmt.Errorf("stop loss must be greater than ask")
		}
		if !order.TakeProfit.IsZero() && order.TakeProfit.Gt(tick.Ask) {
			return nil, fmt.Errorf("take profit must be less than ask")
		}
		positionSide = common.PositionSideShort
		availableLiquidity = tick.BidVolume
	}

	size := order.Size.Sub(order.FilledSize)
	if availableLiquidity.IsZero() {
		return nil, fmt.Errorf("available liquidity is zero")
	}

	if size.Gt(availableLiquidity) {
		size = availableLiquidity
		size = size.Rescale(2)
	}

	s.positionIdCounter++
	return &common.Position{
		Source:        simulatorComponentName,
		Symbol:        order.Symbol,
		ExecutionID:   utility.GetExecutionID(),
		TraceID:       utility.CreateTraceID(),
		OrderTraceIDs: []utility.TraceID{order.TraceID},
		Id:            s.positionIdCounter,
		Status:        positionStatusPendingOpen,
		Side:          positionSide,
		Size:          size,
		StopLoss:      order.StopLoss,
		TakeProfit:    order.TakeProfit,
		Currency:      s.accountCurrency,
		TimeStamp:     s.simulationTime,
	}, nil
}

func (s *Simulator) modifyPosition(order common.Order, tick common.Tick) error {
	for _, position := range s.openPositions {
		if position.Id == order.PositionId {
			if position.Side == common.PositionSideLong {
				if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Gte(order.TakeProfit) {
					return fmt.Errorf("stop loss must be less than take profit")
				}
				if !order.StopLoss.IsZero() && order.StopLoss.Gt(tick.Bid) {
					return fmt.Errorf("stop loss must be less than bid")
				}
				if !order.TakeProfit.IsZero() && order.TakeProfit.Lt(tick.Bid) {
					return fmt.Errorf("take profit must be greater than bid")
				}
			} else {
				if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Lte(order.TakeProfit) {
					return fmt.Errorf("stop loss must be greater than take profit")
				}
				if !order.StopLoss.IsZero() && order.StopLoss.Lt(tick.Ask) {
					return fmt.Errorf("stop loss must be greater than ask")
				}
				if !order.TakeProfit.IsZero() && order.TakeProfit.Gt(tick.Ask) {
					return fmt.Errorf("take profit must be less than ask")
				}
			}
			if !order.StopLoss.IsZero() {
				position.StopLoss = order.StopLoss
			}
			if !order.TakeProfit.IsZero() {
				position.TakeProfit = order.TakeProfit
			}
			position.OrderTraceIDs = append(position.OrderTraceIDs, order.TraceID)
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", order.PositionId)
}

func (s *Simulator) shouldExecuteLimitOrder(order common.Order, tick common.Tick) bool {
	if order.Command == common.OrderCommandPositionOpen {
		if order.Side == common.OrderSideBuy {
			return order.Price.Gte(tick.Ask)
		} else {
			return order.Price.Lte(tick.Bid)
		}
	} else if order.Command == common.OrderCommandPositionClose {
		if order.Side == common.OrderSideBuy {
			return order.Price.Lte(tick.Bid)
		} else {
			return order.Price.Gte(tick.Ask)
		}
	}
	return false
}

func (s *Simulator) shouldClosePosition(position common.Position, tick common.Tick) bool {
	if !strings.EqualFold(position.Symbol, tick.Symbol) {
		return false
	}

	if position.Side == common.PositionSideLong {
		if (!position.TakeProfit.IsZero() && tick.Bid.Gte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Bid.Lte(position.StopLoss)) {
			return true
		}
	} else {
		if (!position.TakeProfit.IsZero() && tick.Ask.Lte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Ask.Gte(position.StopLoss)) {
			return true
		}
	}
	return false
}

func (s *Simulator) calcPositionProfits(position *common.Position, closePrice fixed.Point) {
	symbolInfo, ok := s.symbolsMap[strings.ToUpper(position.Symbol)]
	if !ok {
		panic(fmt.Sprintf("symbol %s not found", position.Symbol))
	}

	exchangeRate := fixed.One
	conversionFeeRate := fixed.Zero
	if s.rateProvider != nil {
		exchangeRate, conversionFeeRate, _ = s.rateProvider.ExchangeRate(s.accountCurrency, symbolInfo.QuoteCurrency, s.simulationTime)
	}

	priceDiff := position.OpenPrice.Sub(closePrice)
	if position.Side == common.PositionSideLong {
		priceDiff = closePrice.Sub(position.OpenPrice)
	}

	if position.Status == common.PositionStatusOpen {
		position.OpenExchangeRate = exchangeRate
		position.OpenConversionFeeRate = conversionFeeRate
		if s.commissionHandler != nil {
			position.Commissions = s.commissionHandler(symbolInfo, *position)
		}
		if !position.OpenConversionFeeRate.IsZero() {
			position.OpenConversionFee = position.Size.Mul(position.OpenExchangeRate).Mul(position.OpenConversionFeeRate)
		}
	}

	if position.Status == common.PositionStatusClosed {
		position.CloseExchangeRate = exchangeRate
		position.CloseConversionFeeRate = conversionFeeRate
		if s.commissionHandler != nil {
			position.Commissions = position.Commissions.Add(s.commissionHandler(symbolInfo, *position))
		}
		if !position.CloseConversionFeeRate.IsZero() {
			position.CloseConversionFee = position.CloseConversionFee.Add(position.Size.Mul(position.CloseExchangeRate).Mul(position.CloseConversionFeeRate))
		}
	}

	daysPassed := int(s.simulationTime.Sub(position.TimeStamp).Hours()) / 24
	if daysPassed > 0 && s.swapHandler != nil {
		for range daysPassed {
			dailySwap := s.swapHandler(symbolInfo, *position)
			position.Swaps = position.Swaps.Add(dailySwap)
		}
	}

	priceDiff = priceDiff.Sub(position.Slippage)
	position.GrossProfit = priceDiff.Mul(position.Size).Mul(symbolInfo.ContractSize).Mul(exchangeRate)
	position.NetProfit = position.GrossProfit.Sub(position.Commissions).Sub(position.Swaps).Sub(position.OpenConversionFee).Sub(position.CloseConversionFee)
	position.Margin = position.Size.Mul(symbolInfo.ContractSize).Mul(closePrice).Mul(exchangeRate).Div(symbolInfo.Leverage)
}

func (s *Simulator) calcFreeMargin() {
	s.freeMargin = s.equity
	for _, position := range s.openPositions {
		s.freeMargin = s.freeMargin.Sub(position.Margin)
	}
}

func (s *Simulator) validateOrder(order common.Order) error {
	if order.Size.Lte(fixed.Zero) {
		return errors.New("order size is zero or negative")
	}

	switch order.Type {
	case common.OrderTypeLimit:
		if err := s.validateLimitOrder(order); err != nil {
			return fmt.Errorf("unable to validate limit order: %w", err)
		}
	case common.OrderTypeMarket:
		if err := s.validateMarketOrder(order); err != nil {
			return fmt.Errorf("unable to validate market order: %w", err)
		}
	default:
		return fmt.Errorf("invalid order type: %d", order.Type)
	}

	switch order.Command {
	case common.OrderCommandPositionOpen:
		if err := s.validatePositionOpenOrder(order); err != nil {
			return fmt.Errorf("unable to validate position open order: %w", err)
		}
	case common.OrderCommandPositionClose:
		if err := s.validatePositionCloseOrder(order); err != nil {
			return fmt.Errorf("unable to validate position close order: %w", err)
		}
	case common.OrderCommandPositionModify:
		if err := s.validatePositionModifyOrder(order); err != nil {
			return fmt.Errorf("unable to validate position modify order: %w", err)
		}
	default:
		return fmt.Errorf("unknown order command: %d", order.Command)
	}

	if err := s.validateStopLossAndTakeProfit(order); err != nil {
		return fmt.Errorf("unable to validate stop loss or take profit: %w", err)
	}

	return nil
}

func (s *Simulator) validateLimitOrder(order common.Order) error {
	if order.Size.IsZero() {
		return fmt.Errorf("order size cannot be zero")
	}
	if order.Price.IsZero() {
		return fmt.Errorf("limit order must have a price")
	}
	return nil
}

func (s *Simulator) validateMarketOrder(order common.Order) error {
	if order.Size.IsZero() {
		return fmt.Errorf("order size cannot be zero")
	}
	if order.TimeInForce == common.TimeInForceGoodTillCancel {
		return fmt.Errorf("market order must not be GTC")
	}
	if order.TimeInForce == common.TimeInForceGoodTillDate {
		return fmt.Errorf("market order must not be GTD")
	}
	return nil
}

func (s *Simulator) validatePositionOpenOrder(order common.Order) error {
	symbolInfo, ok := s.symbolsMap[strings.ToUpper(order.Symbol)]
	if !ok {
		return fmt.Errorf("symbol info for symbol %s is not supported", order.Symbol)
	}

	exchangeRate := fixed.One
	if s.rateProvider != nil {
		var err error
		exchangeRate, _, err = s.rateProvider.ExchangeRate(s.accountCurrency, symbolInfo.QuoteCurrency, s.simulationTime)
		if err != nil {
			return fmt.Errorf("unable to retrieve exchange rate for %s", symbolInfo.QuoteCurrency)
		}
	}

	price := order.Price
	if price.IsZero() {
		tick, ok := s.lastTickMap[strings.ToUpper(order.Symbol)]
		if !ok {
			return fmt.Errorf("unable to determine current asset price")
		}
		price = tick.Bid
		if order.Side == common.OrderSideBuy {
			price = tick.Ask
		}
	}

	requiredMargin := order.Size.Mul(symbolInfo.ContractSize).Mul(price).Mul(exchangeRate).Div(symbolInfo.Leverage)
	availableMarginAfter := s.freeMargin.Sub(requiredMargin)
	availableMarginAfterRate := availableMarginAfter.Div(s.equity).MulInt(100)
	if availableMarginAfterRate.Lte(s.maintenanceMarginRate) {
		return fmt.Errorf("required margin %s exceeds free margin %s", requiredMargin.String(), s.freeMargin.String())
	}
	return nil
}

func (s *Simulator) validatePositionCloseOrder(order common.Order) error {
	if order.PositionId == 0 {
		return fmt.Errorf("position ID required for close order")
	}
	for _, position := range s.openPositions {
		if position.Id == order.PositionId {
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", order.PositionId)
}

func (s *Simulator) validatePositionModifyOrder(order common.Order) error {
	if order.PositionId == 0 {
		return fmt.Errorf("position ID required for modify order")
	}
	for _, position := range s.openPositions {
		if position.Id == order.PositionId {
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", order.PositionId)
}

func (s *Simulator) validateStopLossAndTakeProfit(order common.Order) error {
	tick, ok := s.lastTickMap[strings.ToUpper(order.Symbol)]
	if !ok {
		return fmt.Errorf("no tick found for symbol %s", order.Symbol)
	}
	if order.Side == common.OrderSideBuy {
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Gte(order.TakeProfit) {
			return fmt.Errorf("stop loss must be less than take profit")
		}
		if !order.StopLoss.IsZero() && order.StopLoss.Gt(tick.Bid) {
			return fmt.Errorf("stop loss must be less than bid")
		}
		if !order.TakeProfit.IsZero() && order.TakeProfit.Lt(tick.Bid) {
			return fmt.Errorf("take profit must be greater than bid")
		}
	} else {
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Lte(order.TakeProfit) {
			return fmt.Errorf("stop loss must be greater than take profit")
		}
		if !order.StopLoss.IsZero() && order.StopLoss.Lt(tick.Ask) {
			return fmt.Errorf("stop loss must be greater than ask")
		}
		if !order.TakeProfit.IsZero() && order.TakeProfit.Gt(tick.Ask) {
			return fmt.Errorf("take profit must be less than ask")
		}
	}
	return nil
}

func (s *Simulator) validateTick(tick common.Tick) error {
	_, ok := s.symbolsMap[strings.ToUpper(tick.Symbol)]
	if !ok {
		return fmt.Errorf("tick with symbol %s is not present in the symbols map", tick.Symbol)
	}
	return nil
}

func (s *Simulator) postBalance() {
	balance := common.Balance{
		Source:      simulatorComponentName,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   s.simulationTime,
		Value:       s.balance,
	}
	if err := s.router.Post(bus.BalanceEvent, balance); err != nil {
		slog.Error("unable to post balance event",
			"error", err, "balance", balance)
	}
}

func (s *Simulator) postEquity() {
	equity := common.Equity{
		Source:      simulatorComponentName,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   s.simulationTime,
		Value:       s.equity,
	}
	if err := s.router.Post(bus.EquityEvent, equity); err != nil {
		slog.Error("unable to post equity event",
			"error", err, "equity", equity)
	}
}

func (s *Simulator) postOrderRejected(order common.Order, reason string) {
	rejectOrder := common.OrderRejected{
		Source:        simulatorComponentName,
		ExecutionId:   utility.GetExecutionID(),
		TraceID:       utility.CreateTraceID(),
		TimeStamp:     s.simulationTime,
		OriginalOrder: order,
		Reason:        reason,
	}
	if err := s.router.Post(bus.OrderRejectionEvent, rejectOrder); err != nil {
		slog.Error("unable to post order rejected event",
			"error", err, "order", order)
	}
}

func (s *Simulator) postOrderFilled(order common.Order, positionId common.PositionId) {
	filledOrder := common.OrderFilled{
		Source:        simulatorComponentName,
		ExecutionId:   utility.GetExecutionID(),
		TraceID:       utility.CreateTraceID(),
		TimeStamp:     s.simulationTime,
		OriginalOrder: order,
		PositionId:    positionId,
	}
	if err := s.router.Post(bus.OrderFilledEvent, filledOrder); err != nil {
		slog.Error("unable to post order filled event",
			"error", err, "order", order)
	}
}

func (s *Simulator) postOrderCancel(order common.Order, cancelSize fixed.Point) {
	cancelledOrder := common.OrderCancelled{
		Source:        simulatorComponentName,
		ExecutionId:   utility.GetExecutionID(),
		TraceID:       utility.CreateTraceID(),
		TimeStamp:     s.simulationTime,
		OriginalOrder: order,
		CancelledSize: cancelSize,
	}
	if err := s.router.Post(bus.OrderCancelledEvent, cancelledOrder); err != nil {
		slog.Error("unable to post order cancel event",
			"error", err, "order", order)
	}
}
